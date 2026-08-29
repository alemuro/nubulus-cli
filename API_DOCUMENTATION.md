# Documentació de les APIs de Nubulus Cloud

Aquest document detalla l'arquitectura, els mecanismes d'autenticació, els models de dades i tots els endpoints HTTP de les APIs utilitzades pel [Terraform Provider de Nubulus Cloud (`terraform-provider-nubuluscloud`)](https://github.com/nubulus-network/terraform-provider-nubuluscloud) per a la gestió de **Zones DNS**, **Registres DNS (RRsets)**, **Túnels WireGuard** i **Rutes de Túnels**.

---

## Índex

1. [Endpoints Base i Variables d'Entorn](#1-endpoints-base-i-variables-dentorn)
2. [Autenticació i Autorització (OAuth2 / Zitadel)](#2-autenticació-i-autorització-oauth2--zitadel)
   - [Model de Tokens de Màquina (Application Tokens)](#model-de-tokens-de-màquina-application-tokens)
   - [Scopes Crítics](#scopes-crítics)
   - [Rols i Permisos](#rols-i-permisos)
3. [Format de Resposta i Gestió d'Errors](#3-format-de-resposta-i-gestió-derrors)
   - [Envelop d'Error](#envelop-derror)
   - [Codis d'Error Comuns](#codis-derror-comuns)
4. [API de DNS (`/api/v1`)](#4-api-de-dns-apiv1)
   - [4.1 Gestió de Zones](#41-gestió-de-zones)
     - [Crear o Reclamar Zona (`POST /api/v1/zones`)](#crear-o-reclamar-zona-post-apiv1zones)
     - [Llistar Zones (`GET /api/v1/zones`)](#llistar-zones-get-apiv1zones)
     - [Obtenir Detall de Zona (`GET /api/v1/zones/{zone}`)](#obtenir-detall-de-zona-get-apiv1zoneszone)
     - [Eliminar Zona (`DELETE /api/v1/zones/{zone}`)](#eliminar-zona-delete-apiv1zoneszone)
   - [4.2 Verificació de Domini](#42-verificació-de-domini)
     - [Obtenir Dades del Repte de Verificació (`GET /api/v1/zones/{zone}/verification`)](#obtenir-dades-del-repte-de-verificació-get-apiv1zoneszoneverification)
     - [Executar Comprovació de Verificació (`POST /api/v1/zones/{zone}/verify`)](#executar-comprovació-de-verificació-post-apiv1zoneszoneverify)
   - [4.3 Registres DNS (RRsets)](#43-registres-dns-rrsets)
     - [Llegir Contingut de la Zona / Tots els RRsets (`GET /api/v1/zones/{zone}/rrsets`)](#llegir-contingut-de-la-zona--tots-els-rrsets-get-apiv1zoneszonerrsets)
     - [Crear o Actualitzar un RRset (`PUT /api/v1/zones/{zone}/rrsets/{fqdn}/{type}`)](#crear-o-actualitzar-un-rrset-put-apiv1zoneszonerrsetsfqdntype)
     - [Eliminar un RRset (`DELETE /api/v1/zones/{zone}/rrsets/{fqdn}/{type}`)](#eliminar-un-rrset-delete-apiv1zoneszonerrsetsfqdntype)
   - [4.4 Regles de Validació i Concurrència DNS](#44-regles-de-validació-i-concurrència-dns)
5. [API de Túnels v2 (`/api/v2`)](#5-api-de-túnels-v2-apiv2)
   - [5.1 Gestió de Túnels](#51-gestió-de-túnels)
     - [Crear o Adoptar Túnel (`POST /api/v2/tunnels`)](#crear-o-adoptar-túnel-post-apiv2tunnels)
     - [Obtenir Túnel i Rutes (`GET /api/v2/tunnels/{id}`)](#obtenir-túnel-i-rutes-get-apiv2tunnelsid)
     - [Llistar Túnels (`GET /api/v2/tunnels`)](#llistar-túnels-get-apiv2tunnels)
     - [Rotar Credencials del Túnel (`POST /api/v2/tunnels/{id}/rotate-token`)](#rotar-credencials-del-túnel-post-apiv2tunnelsidrotate-token)
     - [Eliminar Túnel (`DELETE /api/v2/tunnels/{id}`)](#eliminar-túnel-delete-apiv2tunnelsid)
   - [5.2 Gestió de Rutes de Túnel](#52-gestió-de-rutes-de-túnel)
     - [Crear Ruta (`POST /api/v2/tunnels/{tunnel_id}/routes`)](#crear-ruta-post-apiv2tunnelstunnel_idroutes)
     - [Llistar Rutes d'un Túnel (`GET /api/v2/tunnels/{tunnel_id}/routes`)](#llistar-rutes-dun-túnel-get-apiv2tunnelstunnel_idroutes)
     - [Obtenir Ruta (`GET /api/v2/tunnels/{tunnel_id}/routes/{route_id}`)](#obtenir-ruta-get-apiv2tunnelstunnel_idroutesroute_id)
     - [Actualitzar Ruta (`PUT /api/v2/tunnels/{tunnel_id}/routes/{route_id}`)](#actualitzar-ruta-put-apiv2tunnelstunnel_idroutesroute_id)
     - [Eliminar Ruta (`DELETE /api/v2/tunnels/{tunnel_id}/routes/{route_id}`)](#eliminar-ruta-delete-apiv2tunnelstunnel_idroutesroute_id)
   - [5.3 Particularitats i Comportaments de l'API de Túnels](#53-particularitats-i-comportaments-de-lapi-de-túnels)

---

## 1. Endpoints Base i Variables d'Entorn

Les APIs de Nubulus estan dividides en serveis diferenciats darrere d'una capa comuna d'autenticació OIDC (Zitadel):

| Servei | URL Base per Defecte | Variable d'Entorn per Override |
| :--- | :--- | :--- |
| **Identity Provider (IdP)** | `https://idp.nubulusnetwork.es/oauth/v2/token` | `NUBULUS_TOKEN_URL` |
| **API de DNS** | `https://dns.api.nubulusnetwork.es` | `NUBULUS_DNS_ENDPOINT` |
| **API de Túnels** | `https://tunel.api.nubulusnetwork.es` | `NUBULUS_TUNNEL_ENDPOINT` |
| **ID de Projecte per Defecte** | `385111705782321341` | `NUBULUS_PROJECT_ID` |

---

## 2. Autenticació i Autorització (OAuth2 / Zitadel)

### Model de Tokens de Màquina (Application Tokens)

Totes les peticions a les APIs de DNS i Túnels requereixen un token d'accés `Bearer` obtingut mitjançant el flux **OAuth2 Client Credentials** (`grant_type=client_credentials`).

* **Client ID**: Configurat via provider o variable `NUBULUS_CLIENT_ID`.
* **Client Secret**: Configurat via provider o variable `NUBULUS_CLIENT_SECRET`.
* **Auth Style**: Les credencials s'envien al cos del missatge (`AuthStyleInParams` / POST body a Zitadel).
* **Multi-tenancy automàtic**: Les peticions mai no passen cap `account_id` explícit com a paràmetre d'URL o cos. Els serveis de backend resolen el compte i l'organització directament a partir de les claims contingudes al JWT del token d'accés.

### Scopes Crítics

Per sol·licitar el token, s'han d'incloure exactament 4 scopes. Si algun d'aquests scopes falta, el token es generarà però l'API el rebutjarà:

```
openid
urn:zitadel:iam:org:project:id:<PROJECT_ID>:aud
urn:zitadel:iam:user:resourceowner
urn:zitadel:iam:org:projects:roles
```

1. **`openid`**: Scope OIDC estàndard.
2. **`urn:zitadel:iam:org:project:id:<PROJECT_ID>:aud`**: Defineix l'audiència (`aud`) del projecte de Nubulus. Sense aquest scope, `aud` s'estableix al nom d'usuari i es perden les claims de rols.
3. **`urn:zitadel:iam:user:resourceowner`**: Proporciona la claim `resourceowner:id`, mitjançant la qual els serveis mapegen a quin compte/organització de Nubulus pertany la petició.
4. **`urn:zitadel:iam:org:projects:roles`**: (*Nota: "projects" en plural*). Imprescindible per a tokens de màquina (Service Accounts), ja que Zitadel només inclou la claim de rols si se sol·licita explícitament aquest scope.

### Rols i Permisos

Els permisos requerits per executar les operacions a les APIs són:

* **`member`**: Permet consultar informació i crear/modificar/eliminar registres DNS (RRsets), túnels i rutes.
* **`owner`** o **`admin`**: Requerit per crear i eliminar zones DNS completes.

---

## 3. Format de Resposta i Gestió d'Errors

### Envelop d'Error

Quan un endpoint falla (codis HTTP 4xx o 5xx), l'API retorna una estructura JSON estàndard:

```json
{
  "error": "CODI_D_ERROR",
  "message": "Descripció detallada del motiu de l'error"
}
```

### Codis d'Error Comuns

| Codi d'Error (`error`) | HTTP Status Típic | Descripció | Acció Recomanada |
| :--- | :--- | :--- | :--- |
| `NO_ACCOUNT_ROLE` | 403 | El token no disposa de claim de rols. | Assegurar que el token es demana amb el scope `urn:zitadel:iam:org:projects:roles`. |
| `NO_ACCOUNT` | 403 / 404 | L'organització no està vinculada a cap compte de Nubulus. | Comprovar si les credencials pertanyen a un entorn diferent (ex. staging vs prod). |
| `ZONE_NOT_ACTIVE` | 409 | S'intenta escriure registres en una zona pendent de verificació o suspesa. | Completar la verificació del domini abans de crear registres. |
| `UPDATE_PRECONDITION_FAILED`| 409 | Conflicte de concurrència (carrera RFC 2136) en editar un RRset. | Reintentar la petició (el client reintenta 1 cop després d'1 segon). |
| `RRSET_NOT_FOUND` / `NOT_FOUND` | 404 | El recurs sol·licitat no existeix. | Crear el recurs o corregir l'identificador. |
| `INVALID_INPUT` | 400 (o 500) | Petició mal formada (FQDN invàlid, port fora de rang, etc.). | Validar el payload de la petició. |
| `HOSTNAME_CONFLICT` | 409 (o 500) | El hostname de la ruta ja està sent utilitzat per un altre compte a la plataforma. | Els hostnames són globals: cal triar un altre nom o alliberar-lo del compte origen. |
| `QUOTA_EXCEEDED` | 429 (o 500) | S'ha assolit el límit màxim de túnels permesos per al compte. | Destruir túnels en desús per alliberar quota. |
| `TUNNEL_INACTIVE` | 409 (o 500) | Operació no permesa sobre un túnel que no està en estat `active`. | Esperar que el túnel passi a actiu. |

---

## 4. API de DNS (`/api/v1`)

L'API de DNS s'executa a `https://dns.api.nubulusnetwork.es` i proporciona gestió autoritativa de dominis i registres.

### 4.1 Gestió de Zones

#### Crear o Reclamar Zona (`POST /api/v1/zones`)

Reclama una zona per al compte associat al token.

* **URL**: `/api/v1/zones`
* **Mètode**: `POST`
* **Headers**:
  * `Authorization: Bearer <token>`
  * `Content-Type: application/json`
  * `Accept: application/json`
* **Cos de la Petició (Request Body)**:
  ```json
  {
    "name": "example.com"
  }
  ```
* **Comportament**:
  * Si el domini és intern (assignat prèviament al compte per Neodigit), la zona es crea immediatament com a `active` (`source: "neodigit"`).
  * Si és un domini extern (`source: "external"`), la zona es crea en estat `pending_verification` i es retorna un repte de validació TXT. Res no es publica als servidors DNS fins que no es verifiqui.
* **Resposta d'Èxit (201 Created / 200 OK)**:
  ```json
  {
    "zone": {
      "id": "zone_01J6A1B2C3D4E5F6G7H8J9K0L1",
      "name": "example.com",
      "source": "external",
      "status": "pending_verification",
      "account_id": "acct_01J6A...",
      "verified_at": null,
      "reserved_until": "2026-09-01T12:00:00Z",
      "created_at": "2026-08-28T18:30:00Z",
      "created_by": "user_01..."
    },
    "verification": {
      "zone": "example.com",
      "status": "pending_verification",
      "source": "external",
      "required": true,
      "txt_record_host": "_nubulus-challenge.example.com",
      "txt_record_value": "nubulus-verify-9f8a7c6b5e4d3c2b1a",
      "nameservers": [
        "ns1.nubulusnetwork.es.",
        "ns2.nubulusnetwork.es."
      ],
      "reserved_until": "2026-09-01T12:00:00Z",
      "verified_at": null
    }
  }
  ```

#### Llistar Zones (`GET /api/v1/zones`)

* **URL**: `/api/v1/zones`
* **Mètode**: `GET`
* **Resposta d'Èxit (200 OK)**:
  ```json
  {
    "data": [
      {
        "id": "zone_01J6A1B2C3D4E5F6G7H8J9K0L1",
        "name": "example.com",
        "source": "external",
        "status": "active",
        "account_id": "acct_01...",
        "verified_at": "2026-08-28T18:35:00Z",
        "created_at": "2026-08-28T18:30:00Z",
        "created_by": "user_01..."
      }
    ]
  }
  ```
  *(Nota: El llistat global no fa transferència DNS amb el primari, per la qual cosa no inclou serial ni nameservers per motius de rendiment).*

#### Obtenir Detall de Zona (`GET /api/v1/zones/{zone}`)

* **URL**: `/api/v1/zones/{zone}` (on `{zone}` és el nom de la zona, ex. `example.com`, codificat per URL)
* **Mètode**: `GET`
* **Resposta d'Èxit (200 OK)**:
  ```json
  {
    "zone": {
      "id": "zone_01J6A1B2C3D4E5F6G7H8J9K0L1",
      "name": "example.com",
      "source": "external",
      "status": "active",
      "account_id": "acct_01...",
      "verified_at": "2026-08-28T18:35:00Z",
      "created_at": "2026-08-28T18:30:00Z",
      "created_by": "user_01..."
    },
    "serial": 2026082801,
    "nameservers": [
      "ns1.nubulusnetwork.es.",
      "ns2.nubulusnetwork.es."
    ],
    "read_at": "2026-08-28T18:40:00Z",
    "primary_error": "",
    "verification": null
  }
  ```
  *Si el servidor DNS primari no respon o falla la transferència XFR, `primary_error` contindrà el codi corresponent (`XFR_REFUSED`, `XFR_TSIG`, `XFR_TIMEOUT`, `XFR_DISABLED`, `XFR_FAILED`) sense que l'endpoint falli amb 5xx.*

#### Eliminar Zona (`DELETE /api/v1/zones/{zone}`)

Elimina la zona del catàleg i de tots els servidors DNS autoritatius. El domini deixa de resoldre immediatament.

* **URL**: `/api/v1/zones/{zone}`
* **Mètode**: `DELETE`
* **Resposta d'Èxit**: `204 No Content` (sense cos).

---

### 4.2 Verificació de Domini

Per a zones externes, cal demostrar el control del domini abans que la plataforma el publiqui i permeti gestionar registres.

#### Obtenir Dades del Repte de Verificació (`GET /api/v1/zones/{zone}/verification`)

* **URL**: `/api/v1/zones/{zone}/verification`
* **Mètode**: `GET`
* **Resposta d'Èxit (200 OK)**:
  ```json
  {
    "zone": "example.com",
    "status": "pending_verification",
    "source": "external",
    "required": true,
    "txt_record_host": "_nubulus-challenge.example.com",
    "txt_record_value": "nubulus-verify-9f8a7c6b5e4d3c2b1a",
    "nameservers": [
      "ns1.nubulusnetwork.es.",
      "ns2.nubulusnetwork.es."
    ],
    "reserved_until": "2026-09-01T12:00:00Z",
    "verified_at": null
  }
  ```

#### Executar Comprovació de Verificació (`POST /api/v1/zones/{zone}/verify`)

Demana a la plataforma que comprovi immediatament si el registre TXT o la delegació NS és visible a internet.

* **URL**: `/api/v1/zones/{zone}/verify`
* **Mètode**: `POST`
* **Cos**: Buit (`{}` o sense cos).
* **Resposta**: Retorna sempre `200 OK` (un intent fallit **no és un error HTTP**, sinó una resposta amb `verified: false`).
* **Exemple d'èxit**:
  ```json
  {
    "zone": "example.com",
    "status": "active",
    "verified": true,
    "method": "txt",
    "checked_at": "2026-08-28T18:35:10Z"
  }
  ```
* **Exemple d'intent no validat**:
  ```json
  {
    "zone": "example.com",
    "status": "pending_verification",
    "verified": false,
    "reason_code": "TXT_NOT_FOUND",
    "reason": "No TXT record found at _nubulus-challenge.example.com",
    "checked_at": "2026-08-28T18:32:00Z"
  }
  ```
  *Valors possibles de `reason_code`: `TXT_NOT_FOUND`, `TXT_MISMATCH`, `NS_MISMATCH`, `LOOKUP_FAILED`.*

---

### 4.3 Registres DNS (RRsets)

L'API treballa a nivell de **RRset** (conjunt de registres del mateix nom i tipus) conforme a la RFC 2136. Si un subdomini té tres registres tipus `A`, s'administren tots tres com un únic objecte amb una llista de valors.

#### Llegir Contingut de la Zona / Tots els RRsets (`GET /api/v1/zones/{zone}/rrsets`)

* **URL**: `/api/v1/zones/{zone}/rrsets`
* **Mètode**: `GET`
* **Resposta d'Èxit (200 OK)**:
  ```json
  {
    "zone": "example.com",
    "serial": 2026082802,
    "rrsets": [
      {
        "name": "example.com.",
        "type": "SOA",
        "ttl": 3600,
        "values": [
          "ns1.nubulusnetwork.es. hostmaster.nubulusnetwork.es. 2026082802 7200 3600 1209600 300"
        ]
      },
      {
        "name": "example.com.",
        "type": "NS",
        "ttl": 86400,
        "values": [
          "ns1.nubulusnetwork.es.",
          "ns2.nubulusnetwork.es."
        ]
      },
      {
        "name": "www.example.com.",
        "type": "A",
        "ttl": 300,
        "values": [
          "198.51.100.10",
          "198.51.100.11"
        ]
      },
      {
        "name": "app.example.com.",
        "type": "CNAME",
        "ttl": 300,
        "values": [
          "tun-abc.nubulustun.com."
        ]
      }
    ],
    "read_at": "2026-08-28T18:45:00Z"
  }
  ```

#### Crear o Actualitzar un RRset (`PUT /api/v1/zones/{zone}/rrsets/{fqdn}/{type}`)

Insereix o sobreescriu completament el conjunt de valors per al nom i tipus especificats de forma idempotent.

* **URL**: `/api/v1/zones/{zone}/rrsets/{fqdn}/{type}`
  * `{zone}`: Nom de la zona (`example.com`).
  * `{fqdn}`: Nom FQDN normalitzat en minúscules amb punt final (`www.example.com.` o `example.com.` per l'apex), codificat per URL.
  * `{type}`: Tipus de registre en majúscules (`A`, `AAAA`, `CNAME`, `MX`, `TXT`, `SRV`, etc.).
* **Mètode**: `PUT`
* **Cos de la Petició (Request Body)**:
  ```json
  {
    "ttl": 300,
    "values": [
      "198.51.100.10",
      "198.51.100.11"
    ]
  }
  ```
* **Resposta d'Èxit (200 OK)**:
  ```json
  {
    "name": "www.example.com.",
    "type": "A",
    "ttl": 300,
    "values": [
      "198.51.100.10",
      "198.51.100.11"
    ]
  }
  ```

#### Eliminar un RRset (`DELETE /api/v1/zones/{zone}/rrsets/{fqdn}/{type}`)

* **URL**: `/api/v1/zones/{zone}/rrsets/{fqdn}/{type}`
* **Mètode**: `DELETE`
* **Resposta d'Èxit**: `204 No Content`

---

### 4.4 Regles de Validació i Concurrència DNS

1. **Format dels Noms**:
   * Nom de zona: RFC 1123, mínim 2 etiquetes, max 63 caràcters per etiqueta, max 253 caràcters totals.
   * Nom de registre: Admet caràcters de servei segons RFC 8552 (`_` inicial en etiquetes com `_dmarc`, `_acme-challenge`, `_domainkey`) i wildcards (`*.example.com.`).
2. **Restriccions de Tipus de Registres**:
   * **Tipus Prohibits**: DNSSEC gestionat pel servidor (`DNSKEY`, `RRSIG`, `NSEC`, `NSEC3`, `NSEC3PARAM`, `CDS`, `CDNSKEY`), `DNAME`, i pseudo-tipus (`AXFR`, `IXFR`, `ANY`, `OPT`, `TSIG`, `TKEY`).
   * **Apex Gestionat**: Els registres `SOA` i `NS` a l'apex (`example.com.`) estan reservats per la plataforma i no es poden modificar ni esborrar. Els registres `NS` en sub-nivells (subdelegacions) sí que estan permesos.
   * **Regla CNAME**: Un `CNAME` només pot tenir **1 sol valor** i **no pot situar-se a l'apex** de la zona.
3. **Límits de TTL i Valors**:
   * TTL mínim: `60` segons.
   * TTL màxim: `604800` segons (7 dies).
   * Màxim de valors per RRset: `100`.
4. **Control de Concurrència Optimista**:
   * En cas de concurrència d'escriptura, l'API retorna HTTP 409 amb `UPDATE_PRECONDITION_FAILED`. El client ha de refrescar la zona i reintentar l'aplicació.

---

## 5. API de Túnels v2 (`/api/v2`)

L'API de túnels s'executa a `https://tunel.api.nubulusnetwork.es`. Proporciona connectivitat segura mitjançant túnels WireGuard sortints cap a la xarxa de Nubulus i exposició de serveis mitjançant rutes HTTP/HTTPS.

### 5.1 Gestió de Túnels

#### Crear o Adoptar Túnel (`POST /api/v2/tunnels`)

Crea un túnel nou o adopta un túnel existent identificat per `external_id`.

* **URL**: `/api/v2/tunnels`
* **Mètode**: `POST`
* **Cos de la Petició (Request Body)**:
  ```json
  {
    "name": "production-tunnel",
    "external_id": "k8s-cluster-prod-01"
  }
  ```
  *(Ambdós camps són opcionals: si s'envia `{}` es genera un túnel sense etiquetes).*

* **Resposta de Creació d'un Nou Túnel (`201 Created`)**:
  > **MOLT IMPORTANT**: Les credencials `tunnel_token` i `wireguard.interface.private_key` **NOMÉS s'entreguen en aquesta resposta de creació**. Cap endpoint de lectura (`GET`) les tornarà a exposar mai.

  ```json
  {
    "tunnel_id": "tun_01J6XYZ...",
    "tunnel_token": "nub_ttok_secret_abc123456789",
    "tunnel_subdomain": "tun-01j6xyz.nubulustun.com",
    "cname_target": "tun-01j6xyz.nubulustun.com.",
    "instructions": "Configureu el client WireGuard amb les dades següents...",
    "wireguard_ip": "10.128.4.15",
    "wireguard": {
      "interface": {
        "private_key": "aW50ZXJmYWNlX3ByaXZhdGVfa2V5X3NlY3JldA==",
        "address": "10.128.4.15/32",
        "dns": "10.128.0.1"
      },
      "peer": {
        "public_key": "cGVlcl9wdWJsaWNfa2V5X2Zyb21fc2VydmVyCg==",
        "endpoint": "gw1.nubulustun.com:51820",
        "allowed_ips": "10.128.0.0/16",
        "persistent_keepalive": 25
      }
    },
    "adopted": false
  }
  ```

* **Resposta d'Adopció (`200 OK` amb `adopted: true`)**:
  Si s'especifica un `external_id` que ja pertany a un túnel del compte:
  ```json
  {
    "tunnel_id": "tun_01J6XYZ...",
    "tunnel_token": "",
    "tunnel_subdomain": "tun-01j6xyz.nubulustun.com",
    "cname_target": "tun-01j6xyz.nubulustun.com.",
    "wireguard_ip": "10.128.4.15",
    "instructions": "Aquest túnel ja existia i ha estat adoptat.",
    "adopted": true
  }
  ```
  *(Les credencials venen buides per seguretat. Si es necessita un token nou, cal invocar `rotate-token`).*

#### Obtenir Túnel i Rutes (`GET /api/v2/tunnels/{id}`)

* **URL**: `/api/v2/tunnels/{id}`
* **Mètode**: `GET`
* **Resposta d'Èxit (200 OK)**:
  ```json
  {
    "tunnel": {
      "id": "tun_01J6XYZ...",
      "account_id": "acct_01...",
      "user_id": "usr_01...",
      "name": "production-tunnel",
      "external_id": "k8s-cluster-prod-01",
      "tunnel_subdomain": "tun-01j6xyz.nubulustun.com",
      "wireguard_ip": "10.128.4.15",
      "wireguard_public_key": "Y2xpZW50X3B1YmxpY19rZXlfMTIzNDU2Cg==",
      "status": "active",
      "online_status": "online",
      "last_handshake_at": "2026-08-28T18:50:00Z",
      "status_changed_at": "2026-08-28T18:30:00Z",
      "created_at": "2026-08-28T18:30:00Z",
      "updated_at": "2026-08-28T18:50:00Z"
    },
    "routes": [
      {
        "id": "rt_01J6...",
        "tunnel_id": "tun_01J6XYZ...",
        "type": "host",
        "hostname": "app.example.com",
        "path_prefix": "/",
        "upstream_host": "192.168.1.50",
        "upstream_port": 8080,
        "upstream_scheme": "http",
        "strip_prefix": false,
        "enabled": true,
        "priority": 100,
        "created_at": "2026-08-28T18:35:00Z",
        "updated_at": "2026-08-28T18:35:00Z"
      }
    ]
  }
  ```
  *Valors d'`online_status`: `online`, `degraded`, `offline`, `unknown`.*

#### Llistar Túnels (`GET /api/v2/tunnels`)

Endpoint amb suport de paginació i filtre opcional per `external_id`.

* **URL**: `/api/v2/tunnels?limit=100&offset=0` o `/api/v2/tunnels?external_id=k8s-cluster-prod-01`
* **Mètode**: `GET`
* **Resposta d'Èxit (200 OK)**:
  ```json
  {
    "data": [
      {
        "id": "tun_01J6XYZ...",
        "account_id": "acct_01...",
        "user_id": "usr_01...",
        "name": "production-tunnel",
        "external_id": "k8s-cluster-prod-01",
        "tunnel_subdomain": "tun-01j6xyz.nubulustun.com",
        "wireguard_ip": "10.128.4.15",
        "wireguard_public_key": "Y2xpZW50X3B1YmxpY19rZXlfMTIzNDU2Cg==",
        "status": "active",
        "online_status": "online",
        "route_count": 1,
        "created_at": "2026-08-28T18:30:00Z",
        "updated_at": "2026-08-28T18:50:00Z"
      }
    ],
    "limit": 100,
    "offset": 0
  }
  ```

#### Rotar Credencials del Túnel (`POST /api/v2/tunnels/{id}/rotate-token`)

Genera un nou `tunnel_token` per al túnel i invalida immediatament l'anterior.

* **URL**: `/api/v2/tunnels/{id}/rotate-token`
* **Mètode**: `POST`
* **Resposta d'Èxit (200 OK)**:
  ```json
  {
    "tunnel_id": "tun_01J6XYZ...",
    "tunnel_token": "nub_ttok_new_secret_987654321"
  }
  ```

#### Eliminar Túnel (`DELETE /api/v2/tunnels/{id}`)

Elimina el túnel i totes les rutes associades.

* **URL**: `/api/v2/tunnels/{id}`
* **Mètode**: `DELETE`
* **Resposta d'Èxit**: `204 No Content`

---

### 5.2 Gestió de Rutes de Túnel

Les rutes defineixen com el trànsit que arriba a un cert `hostname` (i opcionalment sota un cert `path_prefix`) es redirigeix a través del túnel WireGuard cap a un servei intern (`upstream`).

#### Crear Ruta (`POST /api/v2/tunnels/{tunnel_id}/routes`)

* **URL**: `/api/v2/tunnels/{tunnel_id}/routes`
* **Mètode**: `POST`
* **Cos de la Petició (Request Body)**:
  ```json
  {
    "type": "path",
    "hostname": "api.example.com",
    "path_prefix": "/v1",
    "upstream_host": "10.0.0.5",
    "upstream_port": 3000,
    "upstream_scheme": "http",
    "strip_prefix": true,
    "priority": 50
  }
  ```
  *Camps:*
  * `type`: `"host"` (coincideix amb tot el trànsit del domini) o `"path"` (requereix un prefix).
  * `hostname`: FQDN del domini a servir. **És únic globalment a tota la plataforma Nubulus**.
  * `path_prefix`: `"/"` per rutes `host`; per rutes `path` és obligatori, ha de començar per `"/"` i no pot ser únicament `"/"`.
  * `upstream_host`: Adreça IP o nom d'amfitrió del servei intern accessible des del client del túnel.
  * `upstream_port`: Port de destinació (1 a 65535).
  * `upstream_scheme`: `"http"` o `"https"` (per defecte `"http"`).
  * `strip_prefix`: Si és `true`, elimina el `path_prefix` de la petició abans d'enviar-la a l'upstream.
  * `priority`: Ordre de preferència quan coincideixen múltiples rutes (**menor número guanya**, per defecte `100`).

* **Resposta d'Èxit (201 Created)**:
  ```json
  {
    "id": "rt_01J6ROUTE...",
    "tunnel_id": "tun_01J6XYZ...",
    "type": "path",
    "hostname": "api.example.com",
    "path_prefix": "/v1",
    "upstream_host": "10.0.0.5",
    "upstream_port": 3000,
    "upstream_scheme": "http",
    "strip_prefix": true,
    "enabled": true,
    "priority": 50,
    "created_at": "2026-08-28T19:00:00Z",
    "updated_at": "2026-08-28T19:00:00Z"
  }
  ```

#### Llistar Rutes d'un Túnel (`GET /api/v2/tunnels/{tunnel_id}/routes`)

* **URL**: `/api/v2/tunnels/{tunnel_id}/routes`
* **Mètode**: `GET`
* **Resposta d'Èxit (200 OK)**:
  ```json
  {
    "routes": [
      {
        "id": "rt_01J6ROUTE...",
        "tunnel_id": "tun_01J6XYZ...",
        "type": "path",
        "hostname": "api.example.com",
        "path_prefix": "/v1",
        "upstream_host": "10.0.0.5",
        "upstream_port": 3000,
        "upstream_scheme": "http",
        "strip_prefix": true,
        "enabled": true,
        "priority": 50,
        "created_at": "2026-08-28T19:00:00Z",
        "updated_at": "2026-08-28T19:00:00Z"
      }
    ],
    "total": 1
  }
  ```

#### Obtenir Ruta (`GET /api/v2/tunnels/{tunnel_id}/routes/{route_id}`)

* **URL**: `/api/v2/tunnels/{tunnel_id}/routes/{route_id}`
* **Mètode**: `GET`
* **Resposta d'Èxit (200 OK)**: Retorna l'objecte `Route` individual.

#### Actualitzar Ruta (`PUT /api/v2/tunnels/{tunnel_id}/routes/{route_id}`)

Permet modificar els paràmetres de destinació i comportament d'una ruta existent.
> **Atenció**: Els camps `type`, `hostname` i `path_prefix` són immutables per al cicle de vida de la ruta. Si es volen canviar, cal eliminar la ruta i crear-ne una de nova.

* **URL**: `/api/v2/tunnels/{tunnel_id}/routes/{route_id}`
* **Mètode**: `PUT`
* **Cos de la Petició (Request Body)**: *(Només s'actualitzen els camps enviats)*
  ```json
  {
    "upstream_host": "10.0.0.6",
    "upstream_port": 8080,
    "upstream_scheme": "https",
    "strip_prefix": false,
    "priority": 10,
    "enabled": false
  }
  ```
* **Resposta d'Èxit (200 OK)**: Retorna l'objecte `Route` actualitzat.

#### Eliminar Ruta (`DELETE /api/v2/tunnels/{tunnel_id}/routes/{route_id}`)

* **URL**: `/api/v2/tunnels/{tunnel_id}/routes/{route_id}`
* **Mètode**: `DELETE`
* **Resposta d'Èxit**: `204 No Content`

---

### 5.3 Particularitats i Comportaments de l'API de Túnels

1. **No Edició de Túnels**: Un túnel no es pot editar in-place (`PUT`/`PATCH`). Canviar el `name` o l'`external_id` implica reemplaçar el túnel i generar una nova interfície WireGuard.
2. **Cicle de Creació de Rutes**:
   * Les rutes noves sempre es creen amb `enabled: true`. Per tenir una ruta deshabilitada d'inici, cal crear-la i fer immediatament un `PUT` amb `enabled: false`.
   * En el `POST` de creació, un valor `priority: 0` és interpretat com a valor no definit ("unset") i l'API li assigna el valor per defecte `100`. Per assignar una prioritat `0` real, cal establir-la mitjançant un `PUT` posterior.
3. **CNAME Target**: Perquè el trànsit públic arribi al túnel, cal crear un registre DNS tipus `CNAME` que apunti des del `hostname` de la ruta cap al `cname_target` (o `tunnel_subdomain`) del túnel (ex: `app.example.com. CNAME tun-01j6xyz.nubulustun.com.`).
