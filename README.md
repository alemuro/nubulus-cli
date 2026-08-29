# Nubulus CLI (`nubulus`)

CLI en Go per a interactuar de manera àgil amb les APIs de **DNS** i **Túnels WireGuard** de **Nubulus Cloud**.

---

## Índex

- [Instal·lació i Compilació](#installació-i-compilació)
- [Autenticació i Configuració](#autenticació-i-configuració)
- [Guia d'Ús](#guia-dús)
  - [Exposar Servidors Locals a Internet (`nubulus expose`)](#exposar-servidors-locals-a-internet-nubulus-expose-estil-ngrok)
  - [Instal·lació de Skills per a Agents d'IA (`nubulus agents install`)](#installació-de-skills-per-a-agents-dia-nubulus-agents-install)
  - [Zones DNS (`nubulus zones`)](#1-zones-dns-nubulus-zones)
  - [Registres DNS / RRsets (`nubulus records`)](#2-registres-dns--rrsets-nubulus-records)
  - [Túnels WireGuard (`nubulus tunnels`)](#3-túnels-wireguard-nubulus-tunnels)
  - [Rutes de Túnels (`nubulus routes`)](#4-rutes-de-túnels-nubulus-routes)
- [Formats de Sortida (`--output`)](#formats-de-sortida---output)
- [Documentació Tècnica d'APIs](#documentació-tècnica-dapis)

---

## Instal·lació i Compilació

Requereix Go 1.22+:

```bash
# Compilar el binari local
make build

# O instal·lar-lo a ~/.local/bin/nubulus
make install

# Executar els tests
make test
```

---

## Autenticació i Configuració

L'autenticació utilitza **Application Tokens** (OAuth2 Client Credentials) obtinguts des del panell de Nubulus (`/dashboard/account/tokens`).

Podeu configurar les credencials de tres maneres:

### Opció A: Configuració Interactiva (Recomanat)

Executeu l'assistent interactiu:
```bash
./nubulus config init
```
Us demanarà:
1. **Client ID**: Introducció de text pla.
2. **Client Secret**: Entrada segura i oculta per pantalla (`password mask`).

Això generarà automàticament el fitxer `~/.config/nubulus/config.yaml` amb permisos segurs `0600`:

```yaml
client_id: "EL_VOSTRE_CLIENT_ID"
client_secret: "EL_VOSTRE_CLIENT_SECRET"
token_url: "https://idp.nubulusnetwork.es/oauth/v2/token"
project_id: "385111705782321341"
dns_endpoint: "https://dns.api.nubulusnetwork.es"
tunnel_endpoint: "https://tunel.api.nubulusnetwork.es"
```

### Opció B: Variables d'entorn

```bash
export NUBULUS_CLIENT_ID="el_vostre_client_id"
export NUBULUS_CLIENT_SECRET="el_vostre_client_secret"
```

### Opció C: Flags directes

```bash
./nubulus zones list --client-id <ID> --client-secret <SECRET>
```

---

## Guia d'Ús

### Exposar Servidors Locals a Internet (`nubulus expose` - estil ngrok)

L'ordre `nubulus expose` permet exposar un servei local a internet de manera 100% automatitzada utilitzant la vostra zona DNS delegada i el túnel WireGuard de Nubulus:

```bash
# 1. Exposició ràpida (genera un subdomini aleatori automàtic, ex. swift-fox-42.tun.aleix.cloud)
./nubulus expose 3000

# 2. Amb subdomini concret (ex. myapp.tun.aleix.cloud)
./nubulus expose 3000 --subdomain myapp

# 3. Especificant el túnel i la zona manualment
./nubulus expose 3000 -s test --zone tun.aleix.cloud --tunnel cb781a20-708e-429a-9e2b-cf54e1e81d9d

# 4. Mode permanent / segon pla (no s'esborra en sortir)
./nubulus expose 3000 -s myapp --detach
```

#### Com funciona per sota?
1. **DNS CNAME**: Crea automàticament el registre `myapp.tun.aleix.cloud CNAME cb781a20-708e-429a-9e2b-cf54e1e81d9d.tunel.svc.nubulusnetwork.es.` al DNS de Nubulus.
2. **Ruta de Túnel**: Registra la ruta al túnel per reenviar el trànsit de `myapp.tun.aleix.cloud` cap al port local `127.0.0.1:3000`.
3. **Dashboard interactiu**: Mostra l'estat del túnel i la URL pública.
4. **Neteja automàtica**: Quan premeu `Ctrl+C`, s'elimina automàticament la ruta i el registre CNAME, deixant-ho tot net.

---

### Instal·lació de Skills per a Agents d'IA (`nubulus agents install`)

La CLI inclou la definició incrustada de la skill per a assistents i agents de codificació (com Antigravity o Claude Code). Per instal·lar-la automàticament al directori global de skills (`~/.gemini/config/skills/nubulus/SKILL.md`):

```bash
./nubulus agents install
```

Un cop instal·lada, els agents d'IA sabran automàticament com utilitzar la CLI per a exposar servidors, administrar dominis i crear túnels quan els ho demaneu en llenguatge natural.

Podeu veure la definició de la skill en qualsevol moment amb:
```bash
./nubulus agents show
```

---

### 1. Zones DNS (`nubulus zones`)

```bash
# Llistar totes les zones del compte
./nubulus zones list

# Obtenir detalls d'una zona (serial SOA, nameservers, estat de verificació)
./nubulus zones get example.com

# Crear / reclamar una nova zona
./nubulus zones create example.com

# Consultar el repte de verificació TXT d'una zona externa
./nubulus zones verification example.com

# Llançar la comprovació activa de verificació
./nubulus zones verify example.com

# Eliminar una zona
./nubulus zones delete example.com [--yes]
```

---

### 2. Registres DNS / RRsets (`nubulus records`)

L'API gestiona els registres a nivell de conjunt (**RRset**).

```bash
# Llistar tots els registres d'una zona
./nubulus records list example.com

# Consultar un registre específic
./nubulus records get example.com www A

# Crear o actualitzar un registre A simple
./nubulus records set example.com www A 198.51.100.10 --ttl 300

# Crear o actualitzar múltiples IPs (round-robin)
./nubulus records set example.com api A 198.51.100.10 198.51.100.11 198.51.100.12 --ttl 300

# Crear un registre CNAME apuntant al subdomini d'un túnel
./nubulus records set example.com app CNAME tun-01j6xyz.nubulustun.com.

# Crear un registre MX a l'apex (@)
./nubulus records set example.com @ MX "10 mail.example.com."

# Eliminar un registre
./nubulus records delete example.com www A [--yes]
```

---

### 3. Túnels WireGuard (`nubulus tunnels`)

```bash
# Llistar túnels
./nubulus tunnels list
./nubulus tunnels list --external-id k8s-prod-01

# Crear un túnel i desar la configuració llesta per a WireGuard
./nubulus tunnels create --name "k8s-ingress" --external-id "k8s-prod-01" --save-wg wg0.conf

# Connectar-se amb WireGuard (a la vostra màquina / servidor)
sudo wg-quick up ./wg0.conf

# Consultar estat del túnel (inclou online_status i rutes)
./nubulus tunnels get tun_01J6XYZ

# Rotar credencials d'un túnel
./nubulus tunnels rotate-token tun_01J6XYZ

# Eliminar un túnel i totes les seves rutes
./nubulus tunnels delete tun_01J6XYZ [--yes]
```

---

### 4. Rutes de Túnels (`nubulus routes`)

Les rutes indiquen quin trànsit públic s'envia a través del túnel cap als serveis locals/interns.

```bash
# Llistar rutes d'un túnel
./nubulus routes list tun_01J6XYZ

# Obtenir detall d'una ruta
./nubulus routes get tun_01J6XYZ rt_01J6ABC

# Crear una ruta de domini complet (host)
./nubulus routes create tun_01J6XYZ \
  --hostname app.example.com \
  --upstream-host 127.0.0.1 \
  --upstream-port 8080 \
  --upstream-scheme http

# Crear una ruta basada en prefix (path) eliminant el prefix cap a l'upstream
./nubulus routes create tun_01J6XYZ \
  --type path \
  --hostname api.example.com \
  --path-prefix /v1 \
  --upstream-host 10.0.0.5 \
  --upstream-port 3000 \
  --strip-prefix \
  --priority 50

# Modificar paràmetres d'una ruta
./nubulus routes update tun_01J6XYZ rt_01J6ABC --upstream-port 9090 --enabled false

# Eliminar una ruta
./nubulus routes delete tun_01J6XYZ rt_01J6ABC [--yes]
```

---

## Formats de Sortida (`--output`)

Totes les ordres admeten el flag global `-o` / `--output`:

- `table` (per defecte, taules estilitzades en text pla).
- `json` (JSON identat per a integració amb scripts o `jq`).
- `yaml` (YAML estructurat).

Exemple:
```bash
./nubulus zones list -o json | jq '.[].name'
```

---

## Documentació Tècnica d'APIs

Per a una referència detallada dels esquemes JSON, payloads de petició/resposta, codis d'error HTTP i comportament intern de les APIs de Nubulus, consulteu [**`API_DOCUMENTATION.md`**](./API_DOCUMENTATION.md).
