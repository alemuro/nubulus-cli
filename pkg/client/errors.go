package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	CodeNoAccountRole          = "NO_ACCOUNT_ROLE"
	CodeNoAccount              = "NO_ACCOUNT"
	CodePreconditionFailed     = "UPDATE_PRECONDITION_FAILED"
	CodeZoneNotActive          = "ZONE_NOT_ACTIVE"
	CodeNotFound               = "NOT_FOUND"
	CodeInvalidInput           = "INVALID_INPUT"
	CodeHostnameConflict       = "HOSTNAME_CONFLICT"
	CodeQuotaExceeded          = "QUOTA_EXCEEDED"
	CodeTunnelInactive         = "TUNNEL_INACTIVE"
)

// APIError represents an error returned by the Nubulus API.
type APIError struct {
	Status  int    `json:"status"`
	Code    string `json:"error"`
	Message string `json:"message"`

	Method string `json:"method"`
	URL    string `json:"url"`
}

func (e *APIError) Error() string {
	switch {
	case e.Code != "":
		return fmt.Sprintf("%s %s: HTTP %d (%s): %s", e.Method, e.URL, e.Status, e.Code, e.Message)
	case e.Message != "":
		return fmt.Sprintf("%s %s: HTTP %d: %s", e.Method, e.URL, e.Status, e.Message)
	default:
		return fmt.Sprintf("%s %s: HTTP %d", e.Method, e.URL, e.Status)
	}
}

// TransportError represents a network failure before receiving an HTTP response.
type TransportError struct {
	Method string
	URL    string
	Err    error
}

func (e *TransportError) Error() string {
	return fmt.Sprintf("%s %s: network error: %v", e.Method, e.URL, e.Err)
}

func (e *TransportError) Unwrap() error { return e.Err }

func parseAPIError(method, url string, resp *http.Response) error {
	apiErr := &APIError{Status: resp.StatusCode, Method: method, URL: url}

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil || len(raw) == 0 {
		return apiErr
	}

	var envelope struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(raw, &envelope); err == nil && envelope.Error != "" {
		apiErr.Code = envelope.Error
		apiErr.Message = envelope.Message
		return apiErr
	}

	apiErr.Message = strings.TrimSpace(string(raw))
	return apiErr
}

// StatusOf returns the HTTP status code of err, or 0 if not an APIError.
func StatusOf(err error) int {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.Status
	}
	return 0
}

// CodeOf returns the Nubulus API error code string, or "".
func CodeOf(err error) string {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.Code
	}
	return ""
}

// IsNotFound returns true if error is HTTP 404.
func IsNotFound(err error) bool { return StatusOf(err) == http.StatusNotFound }

// IsConflict returns true if error is HTTP 409.
func IsConflict(err error) bool { return StatusOf(err) == http.StatusConflict }

// IsLostRace returns true if error is UPDATE_PRECONDITION_FAILED.
func IsLostRace(err error) bool {
	return IsConflict(err) && CodeOf(err) == CodePreconditionFailed
}

// FriendlyExplanation formats error into a human actionable description.
func FriendlyExplanation(action string, err error) string {
	if err == nil {
		return ""
	}

	var transportErr *TransportError
	if errors.As(err, &transportErr) {
		return fmt.Sprintf("No s'ha pogut connectar amb el servidor a %s (%s). Comproveu la connexió de xarxa.", transportErr.URL, transportErr.Err)
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return err.Error()
	}

	switch apiErr.Code {
	case "ACCOUNTS_UNREACHABLE":
		return "El servei intern de comptes de Nubulus no ha respost a temps (HTTP 503 ACCOUNTS_UNREACHABLE). Sol ser una incidència transitòria de xarxa o sobrecàrrega al backend de Nubulus. La CLI ho reintenta automàticament."
	case CodeNoAccountRole:
		return "El token no té rols assignats. Assegureu-vos d'haver generat un Application Token vàlid a la plataforma amb permisos."
	case CodeNoAccount:
		return "El token és vàlid però l'organització no està associada a cap compte de Nubulus actiu."
	case CodeZoneNotActive:
		return "La zona encara no està activa (està pendent de verificació o suspesa). Cal verificar el domini abans de crear registres."
	case CodeInvalidInput:
		return fmt.Sprintf("Dades de petició invàlides: %s", apiErr.Message)
	case CodeHostnameConflict:
		return "El hostname ja està utilitzat per un altre compte a la plataforma Nubulus (els hostnames són globals)."
	case CodeQuotaExceeded:
		return "S'ha assolit el límit màxim de túnels permesos per al compte."
	case CodeTunnelInactive:
		return "El túnel no està actiu. Les rutes només es poden afegir o modificar en túnels actius."
	case CodePreconditionFailed:
		return "Conflicte d'escriptura (algú ha modificat el registre simultàniament). Reintenteu l'operació."
	}

	if apiErr.Status == http.StatusForbidden {
		return fmt.Sprintf("Accés denegat (HTTP 403): %s. Comproveu els permisos del token.", apiErr.Message)
	}
	if apiErr.Status == http.StatusNotFound {
		return fmt.Sprintf("Recurs no trobat (HTTP 404): %s", apiErr.Message)
	}

	return apiErr.Error()
}
