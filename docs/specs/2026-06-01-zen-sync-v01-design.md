# zen-sync v0.1 — Design

**Data:** 2026-06-01
**Autor:** Gustavo Guarda
**Status:** Aprovado (aguardando spec review)
**Antecedentes:** [`~/.dotfiles/docs/superpowers/specs/2026-06-01-zen-syncthing-design.md`](../../../.dotfiles/docs/superpowers/specs/2026-06-01-zen-syncthing-design.md)
(spec do experimento shell-script dentro do dotfiles, base validada para este v0.1)

## Contexto

Zen Browser não tem continuidade tipo Arc entre máquinas. Firefox Sync cobre
bookmarks/history/extensions/senhas, mas **workspaces e estado de abas — o que
diferencia Zen do Firefox vanilla — ficam de fora**. É a reclamação mais comum
de Arc-refugees migrando pro Zen.

Tentativa anterior (`~/Projects/zen-sync2`, ~5400 linhas TS como extensão pura)
provou ser frágil: dependia de APIs experimentais do Zen (`browser.zen.*`) que
quebram a cada release, e nunca conseguia ler o estado real sem decifrar
formato interno do Zen.

Iteração subsequente (`~/.dotfiles/scripts/zen-sync/`, ~150 linhas shell)
provou que **sincronizar arquivos é suficiente**: `zen-sessions.jsonlz4` no
profile carrega workspaces + abas + essentials + pinned num único blob binário.
Cópia byte-a-byte resolve sem decodificar nada. Round-trip validado em 2 Macs
(host + VM) com hashes idênticos.

Este v0.1 transforma o pattern shell-script em um produto distribuível como
open source, mantendo a simplicidade do design e adicionando o que o shell não
faz: rodar em background sem o user precisar fechar o browser pra sincronizar.

## Objetivo

Produto open source, single-binary (Go), distribuído via Homebrew, que dá ao
Zen Browser no macOS a sensação Arc-like de continuidade entre máquinas:
trabalhe no Mac A com o Zen aberto, vá pro Mac B, clique no launcher, encontre
exatamente onde parou.

## Não-objetivos

- **Bookmarks/history/extensions/senhas** — Firefox Sync já cobre.
- **Sync com Zen rodando nos dois Macs simultâneo** — last-write-wins
  documentado; not solving CRDT/real-time merge.
- **Reimplementar transporte** — user traz a pasta sincronizada (Syncthing,
  iCloud Drive, Dropbox, USB, o que for). Helper opera sobre filesystem.
- **Extensão WebExt** — fica pra v0.2 ou depois. v0.1 é helper-only.
- **Linux/Windows** — v0.1 é macOS-only. Cross-platform vira issue/PR.
- **Cripto E2E própria** — transport-dependent (Syncthing/iCloud já são E2E).
- **Multi-profile Zen** — assume único profile `*.Default (release)`.
- **GUI / menu bar app** — CLI + Zen Sync.app launcher cobrem o uso. Menu bar
  vira v0.3+.

## Audiência

- **Pessoal**: Gustavo, em 2 Macs.
- **Comunidade**: Arc-refugees power-user no Zen Browser macOS. Confortáveis
  com `brew install`. Reclamando publicamente em fóruns/Reddit/Discord do Zen
  da falta de sync de sessions.

## Arquitetura

Três componentes, **um único binário Go** com subcomandos:

```
┌──────────────────────────────────────────────────────────┐
│ /Applications/Zen Sync.app   (launcher visual no Dock)   │
│   └─ chama internamente: zen-sync open                   │
├──────────────────────────────────────────────────────────┤
│ zen-sync daemon  (LaunchAgent, sempre rodando)           │
│   ├─ fsnotify watch nos arquivos sync                    │
│   ├─ debounce 1s pós-event (evita flush parcial)         │
│   ├─ hash-check (evita push redundante)                  │
│   └─ cp → sync_dir + stamp last-push-host                │
├──────────────────────────────────────────────────────────┤
│ zen-sync CLI  (subcomandos one-shot)                     │
│   ├─ init        setup interativo                        │
│   ├─ open        pull-then-launch Zen                    │
│   ├─ push|pull   manual (debug/recovery)                 │
│   ├─ status      última sync, hashes, last-push-host     │
│   ├─ restore     lista + restaura backups                │
│   └─ doctor      diagnóstico abrangente                  │
└──────────────────────────────────────────────────────────┘
```

Tudo mesmo executável (`os.Args[0] + os.Args[1]` dispatcher). Mantém deploy e
update simples — uma única blob pra atualizar.

## Componentes em detalhe

### Daemon

- **Trigger**: LaunchAgent `RunAtLoad=true` + `KeepAlive=true`. Inicia no login
  do user, reinicia se cair.
- **Watch**: `fsnotify.NewWatcher()` em cada arquivo de `config.files` dentro
  do `zen_profile`. Não watch o dir inteiro — só os arquivos específicos
  (evita ruído de cache do Firefox).
- **Debounce**: ao receber evento `Write`/`Create`/`Rename`, espera 1s. Se
  novo evento chegar nesse intervalo, reseta o timer. Quando o timer expira,
  ler o arquivo e hashear.
- **Hash-check**: SHA-256 do arquivo atual vs último hash visto. Igual = skip
  (write redundante; Firefox às vezes regrava sem mudança real).
- **Push**: `cp -a` (preservando mtime/permissões) do arquivo pro `sync_dir`,
  atualiza `sync_dir/.meta/last-push-host` com hostname.
- **Lifecycle**: sem `init` próprio. Lê config no startup. Reconfigura se
  SIGHUP. Sai limpo no SIGTERM (LaunchAgent envia).

### Launcher (Zen Sync.app)

- **Estrutura**: `.app` bundle minimalista (`Contents/Info.plist` +
  `Contents/MacOS/launcher` shell stub que faz `exec /opt/homebrew/bin/zen-sync open`).
- **Ícone**: mesmo do Zen (copiado do `/Applications/Zen.app` no `init`),
  diferenciado só por um badge sutil no canto (opcional v0.1.x).
- **Bundle ID**: `io.github.gustavoguarda.zen-sync.launcher`.
- **Geração**: `zen-sync init` cria via `osacompile -o` ou template estático.

### CLI

Subcomandos (cobertura no v0.1):

| Comando | Comportamento |
|---|---|
| `zen-sync init` | Wizard: pergunta `sync_dir`, valida, autodetecta `zen_profile`, escreve `~/.config/zen-sync/config.toml`, instala LaunchAgent, instala Zen Sync.app, prompt "use este Mac como source-of-truth?" → primeiro push. |
| `zen-sync open` | Pull (com lógica de skip-if-last-push-host) → `open -W -na Zen "$@"`. Forwarda args pro Zen. |
| `zen-sync push [--dry-run]` | Push manual. Útil pra debug, e pra "forçar source-of-truth" sem esperar daemon. |
| `zen-sync pull [--force] [--dry-run]` | Pull manual. `--force` ignora skip. |
| `zen-sync status` | Tabela: cada arquivo sync, hash local, hash sync, mtime local, mtime sync, last-push-host, daemon running? |
| `zen-sync restore [<backup_name>]` | Sem arg: lista backups. Com arg: restaura. |
| `zen-sync doctor` | Diagnóstico: profile path? daemon up? sync_dir writable? hashes alinhados? Zen rodando? |
| `zen-sync version` | Versão + commit SHA + build date. |
| `zen-sync uninstall` | Remove LaunchAgent, remove Zen Sync.app. **Não toca** config nem sync_dir nem backups (defesa em profundidade). |

## Fluxos de dados

### Push (Zen rodando)

```
Zen flush periódico (~15s) → grava zen-sessions.jsonlz4
       ↓
fsnotify Write event no daemon
       ↓
Debounce 1s (evita ler durante flush)
       ↓
SHA-256(arquivo) == últimoHashVisto?
       ├─ sim: skip (write redundante)
       └─ não: cp pra sync_dir/, atualiza last-push-host, log
                        ↓
                 Transport externo (Syncthing/iCloud/Dropbox) replica
```

### Pull + launch (click no Zen Sync.app)

```
User clica Zen Sync.app no Dock
       ↓
zen-sync open
       ↓
Zen já rodando?  (pgrep -f "/Zen.app/Contents/MacOS/")
       ├─ sim: pula pull, open -a Zen traz pra frente, exit 0
       └─ não:
           ↓
       last-push-host == este hostname?  (sem --force)
           ├─ sim: pula pull (estado já é nosso)
           └─ não:
               ↓
           Pra cada arquivo em config.files:
             1. cp local → backup_dir/<file>.<ts>
             2. cp sync_dir/<file> → local (skip se não existe no sync)
           ↓
           open -W -na Zen           # bloqueia até fechar
           ↓
           (durante uso, daemon segue empurrando)
           ↓
           (close do Zen dispara último flush → daemon push)
```

### Ambos os Macs com Zen aberto (cenário hostil)

- Daemon de cada Mac empurra independentemente.
- `sync_dir` recebe writes intercalados de ambos.
- Última escrita ganha por arquivo (last-write-wins).
- `zen-sync doctor` detecta condição: "sync_dir.last-push-host mudou X vezes
  em Y minutos" → warning.
- Documentado no README como cenário **não suportado**. Workaround: feche
  num antes de abrir no outro.

## Configuração

Arquivo: `~/.config/zen-sync/config.toml` (criado por `zen-sync init`):

```toml
# Mandatory
sync_dir = "~/BrowserSync/Zen"

# Optional (auto-detected on init; override manualmente se necessário)
zen_profile = "~/Library/Application Support/zen/Profiles/abc.Default (release)"
zen_running_pattern = "/Zen.app/Contents/MacOS/"

# Arquivos sincronizados. Default razoável; user pode extender se quiser.
files = [
  "zen-sessions.jsonlz4",
  "zen-live-folders.jsonlz4",
  "containers.json",
]

[daemon]
debounce_ms = 1000
log_level = "info"

[backup]
keep = 5
```

Paths expandidos: `~/` → `$HOME` em load-time.

## Paths fixos

| Item | Path |
|---|---|
| Config | `~/.config/zen-sync/config.toml` |
| Logs do daemon | `~/Library/Logs/zen-sync/daemon.log` (rotação em 1MB, mantém 3) |
| Logs do CLI (init, open, etc) | `~/Library/Logs/zen-sync/cli.log` |
| Backups | `~/.local/state/zen-sync/backups/<file>.<ts>` (últimos 5 por tipo) |
| LaunchAgent plist | `~/Library/LaunchAgents/io.github.gustavoguarda.zen-sync.daemon.plist` |
| Launcher .app | `/Applications/Zen Sync.app` |
| Binário | `/opt/homebrew/bin/zen-sync` (arm64) ou `/usr/local/bin/zen-sync` (intel) |

## Arquivos sincronizados

| Arquivo | Conteúdo | Formato |
|---|---|---|
| `zen-sessions.jsonlz4` | Workspaces + tabs + pinned + essentials | Mozilla LZ4 (cópia binária) |
| `zen-live-folders.jsonlz4` | Live folders | Mozilla LZ4 (cópia binária) |
| `containers.json` | Contextual identities | JSON puro |

**Não sincronizado**: tudo Firefox-padrão (`logins.db`, `cookies.sqlite`,
`places.sqlite`, `key4.db`, `weave/`), `zen-sessions-backup/`,
`sessionstore-backups/`, `prefs.js`, extensões, addons.

## Detecção do Zen

- **Profile**: glob `~/Library/Application Support/zen/Profiles/*.Default (release)`.
  Fallback `*.Default`. `$ZEN_PROFILE` override.
- **Processo rodando**: `pgrep -f "/Zen.app/Contents/MacOS/"`.
  Override por `ZEN_RUNNING_PATTERN` no config.

## Conflict resolution

| Cenário | Comportamento |
|---|---|
| Mac A trabalha, B fechado | A empurra contínuo; B abre via launcher e pega estado. ✅ |
| Mac A trabalha, B abre via launcher | Pull no B, B vira last-push-host. Daemon do A continua empurrando; nas próximas flushes, sync_dir recebe writes do A; `doctor` no B mostra "remote updated since your pull". |
| Ambos abertos editando | Last-write-wins por arquivo. Daemon ainda evita push redundante via hash-check. `doctor` detecta e avisa. |
| Mac novo, sync_dir vazio | Primeiro push do mac existente popula; pull no mac novo restaura. |

## Estrutura do repositório

```
zen-sync/
├── README.md                 # pitch + quick install
├── LICENSE                   # MIT
├── CHANGELOG.md
├── go.mod
├── go.sum
├── Makefile                  # test, build, install-local, clean
├── .goreleaser.yaml          # release config (darwin arm64+amd64)
├── .golangci.yaml            # lint config
├── .github/
│   ├── workflows/
│   │   ├── ci.yml            # test + vet + lint em PR
│   │   └── release.yml       # GoReleaser on tag push
│   └── ISSUE_TEMPLATE/
│       ├── bug.yaml
│       └── feature.yaml
├── cmd/
│   └── zen-sync/
│       └── main.go           # entrypoint, subcommand dispatch
├── internal/
│   ├── config/               # TOML load, defaults, validation, ~/ expand
│   ├── profile/              # detect_zen_profile, zen_running
│   ├── sync/                 # push, pull, hash, backup rotation
│   ├── daemon/               # fsnotify watcher + debounce loop
│   ├── launcher/             # gera/instala Zen Sync.app
│   ├── plist/                # gera/instala/remove LaunchAgent
│   ├── cli/                  # subcommand handlers
│   └── logger/               # rotating file logger
├── docs/
│   ├── specs/
│   │   └── 2026-06-01-zen-sync-v01-design.md
│   ├── INSTALL.md            # brew + init + sync_dir options (Syncthing/iCloud/etc)
│   ├── ARCHITECTURE.md       # high-level (link pra spec)
│   └── TROUBLESHOOTING.md    # doctor outputs explicados + recovery
└── testdata/
    └── fake-profile/         # arquivos sintéticos pra tests
```

## Distribuição

- **GitHub Releases via GoReleaser**: tag `v0.1.0` dispara workflow que
  produz `zen-sync_v0.1.0_darwin_arm64.tar.gz` + `_darwin_amd64.tar.gz` +
  checksums SHA-256 + SBOM CycloneDX. Tudo signed se o user tiver setup.
- **Homebrew tap**: repo `gustavoguarda/homebrew-zen-sync` com Formula
  apontando pros releases. Install: `brew install gustavoguarda/zen-sync/zen-sync`.
- **Manual install fallback**: `curl -L .../zen-sync_v0.1.0_darwin_arm64.tar.gz | tar -xz -C /usr/local/bin`.
- **No App Store / no notarization** no v0.1. Doc adiciona "Allow Apps from
  Identified Developers" workaround se Gatekeeper reclamar.

## Migração do estado atual

| Lugar | Ação |
|---|---|
| `~/Projects/zen-sync2` | **Não tocar.** Mantido local como backup/referência da tentativa de extensão. |
| `~/Projects/zen-sync` | **Novo repo.** Onde o v0.1 vive. |
| `~/.dotfiles/scripts/zen-sync/` | Manter durante migração. Quando v0.1 estiver estável por 1 semana, remover do dotfiles + atualizar `aliases/zen.zsh` pra apontar pro binário Go. |
| `~/.dotfiles/docs/superpowers/{specs,plans}/2026-06-01-zen-syncthing*` | Manter como histórico do experimento. |
| `~/BrowserSync/Zen/` | **Manter intacto.** Daemon Go grava nos mesmos arquivos, mesmo `last-push-host`. Migração transparente. |
| Syncthing config | **Manter intacto.** É transporte externo, não muda. |

## Critérios de v0.1 pronto

- [ ] `brew install gustavoguarda/zen-sync/zen-sync` funciona em darwin arm64 e amd64.
- [ ] `zen-sync init` cria config, instala LaunchAgent, instala Zen Sync.app,
      faz primeiro push.
- [ ] Daemon detecta mudanças em `zen-sessions.jsonlz4` via fsnotify e empurra
      em < 2s (debounce 1s + cp).
- [ ] Zen Sync.app no Dock faz pull-then-launch transparente.
- [ ] `zen-sync doctor` reporta cada falha estruturada (profile, sync_dir,
      daemon, hash drift, Zen running).
- [ ] `zen-sync status` mostra última push, hash por arquivo, last-push-host.
- [ ] `zen-sync restore` lista backups e restaura.
- [ ] `zen-sync uninstall` remove LaunchAgent + .app sem tocar config/sync_dir/backups.
- [ ] Smoke test em Go que faz round-trip sintético (mirror do shell smoke-test).
- [ ] Unit tests cobrem: config parse, profile detect, hash, backup rotation, debounce.
- [ ] CI verde: `go test ./...`, `go vet`, `golangci-lint`.
- [ ] README com pitch + GIF demo (gravado no asciinema ou QuickTime).
- [ ] INSTALL.md cobre Syncthing, iCloud Drive, Dropbox como `sync_dir`.
- [ ] TROUBLESHOOTING.md cobre cada output do `doctor` com recovery step.
- [ ] LICENSE (MIT) presente.

## Fora do escopo (v0.2+)

- Extensão WebExt (settings UI, notificações in-browser, manual sync button).
- Suporte Linux + Windows.
- Menu bar app (status contínuo visual).
- Conflict UI sofisticada (diff visual de workspaces antes de aplicar).
- Sync de bookmarks/history (Firefox Sync já cobre; mexer em `places.sqlite` é risco).
- Cripto E2E própria (Syncthing/iCloud já é E2E; Gist precisaria — fora do escopo).
- Multi-profile Zen.
- Sync entre OSes diferentes (compatibilidade do arquivo não testada).
- Notarização Apple.

## Riscos conhecidos

1. **Update do Zen muda o nome do arquivo** (`zen-sessions.jsonlz4` → outra
   coisa). Mitigação: `config.files` é override; user atualiza o config.
   Doctor detecta arquivo ausente.
2. **Path do profile com sufixo diferente** (`.Default` vs `.Default (release)`
   vs nightly). Mitigação: fallback glob + `zen_profile` override.
3. **Process name varia entre builds**. Mitigação: pattern por path do bundle
   + `zen_running_pattern` override.
4. **Gatekeeper bloqueia Zen Sync.app sem notarização**. Mitigação: doc com
   "right-click → Open" workaround; futuro notarization se demanda crescer.
5. **fsnotify perde eventos sob carga extrema** (raro no use case real
   mas conhecido em macOS). Mitigação: poll periódico de 30s como heartbeat
   secundário; hash-check pega divergências.
6. **`~/BrowserSync/Zen/` exposta a outros users do Mac**. Mitigação:
   `chmod 700` no `init` (parent + child).
7. **Helper roda mas Zen não foi instalado** — daemon detecta no startup,
   loga erro fatal mas re-tenta a cada 30s; LaunchAgent não fica em crash loop.

## Decisões deliberadas

- **Single binary com subcomandos**: deploy + update simples. Daemon, CLI e
  launcher compartilham código.
- **BYO sync_dir**: user já tem Syncthing/iCloud/Dropbox; reimplementar
  transport é o erro do zen-sync2. Doc recomenda Syncthing.
- **fsnotify em vez de polling**: latência menor, CPU mínimo idle. Poll
  secundário cobre eventos perdidos.
- **Hash-check antes de push**: Firefox às vezes regrava sem mudança;
  evitamos tráfego inútil + churn no last-push-host.
- **`.app` launcher em vez de interceptar Zen.app**: simples, robusto, sem
  race conditions. User aceita "tem que arrastar isso pro Dock".
- **Helper-only v0.1, extensão como v0.2+**: ship simples primeiro. Audiência
  Zen Browser é tech-comfortable o suficiente pra `brew install`.
- **MIT license**: máxima adoção, padrão da indústria pra dev tools, sem
  fricção legal.
- **macOS-only v0.1**: focar onde já temos validação; Linux/Windows como
  contribuições externas ou v0.2+.
- **Não tocar `~/Projects/zen-sync2`**: mantém histórico/aprendizado local.

## Follow-ups pós v0.1

- v0.1.x patches: ícone customizado pro launcher, notarização, melhor copy
  no `init` wizard.
- v0.2 — **Extensão WebExt opcional**: settings UI, "sync now" button,
  notificação in-browser. Comunica com daemon via native messaging.
- v0.3 — **Linux + systemd unit**. PR-friendly se alguém aparecer.
- v0.4 — **Menu bar app** (Swift ou Cocoa via Go bindings). Status
  contínuo, "force sync" button.
- v0.x — Multi-profile Zen, sync de mais arquivos opcionais.

## Definição de sucesso

- **Pessoal**: 1 mês de uso sem precisar abrir terminal pra sincronizar nada
  fora do `init` inicial.
- **Comunidade**: 10 stars no GitHub + 1 issue de Arc-refugee em 3 meses.
  Não precisa virar fenômeno.
- **Manutenção**: ≤ 1h/mês após ship do v0.1. Se passar, algo no design
  ficou frágil.
