# PLAN.md — tinywasm/orm Naming Refactoring

Master orchestrator for renaming the code generator output and clarifying its CLI name.

---

## Development Rules

- **Testing Runner (`gotest`):** ALWAYS use the globally installed `gotest` CLI command.
- **Documentation First:** Update docs before writing code.
- **Explicit Execution:** Never start writing/modifying actual codebase source code unless explicitly told to execute.

---

## Architecture decisions: CLI Name and File Suffix

### 1. Nombre de la herramienta CLI (`orm` vs `ormc`)
Se recomienda **mantener el nombre `ormc`** (o en su defecto renombrarlo a `ormgen`).
**Justificación:**
- **Convenciones de Ecosistema:** En Go y herramientas modernas, los binarios que compilan o generan código a partir de definiciones usan la convención `c` de "compiler" (ej. `protoc` para Protobuf, `sqlc` para SQL) o el sufijo `gen` (ej. `mockgen`).
- **Separación de Responsabilidades:** `orm` es fundamentalmente la **librería de runtime** (el paquete importado en el código). Si llamamos al CLI también `orm` (ej. ejecutando `//go:generate orm`), creamos una sobrecarga cognitiva. Hablaríamos del "orm" sin saber si nos referimos a la fase de compilación o ejecución. `ormc` separa físicamente la "herramienta de compilación AST" de la "librería de mapeo".
- **Coherencia del Sufijo Generado:** Que `ormc` genere archivos `_orm.go` es consistente. `ormc` genera la "capa de integración con el framework ORM", por lo que el archivo resultante le rinde homenaje al framework (`_orm.go`), de la misma manera que `protoc` genera archivos `_pb.go` (por protocol buffers) y no `_protoc.go`.

### 2. Sufijo del archivo generado (`_db.go` → `_orm.go`)
Se debe deshacer la decisión previa (`_db.go`) y volver a **`_orm.go`** para todo output autogenerado por la CLI.
**Justificación:**
- Como vimos en `appointment-booking`, el sufijo `_db.go` es demasiado genérico. Invita a que los desarrolladores creen archivos manuales como `model_orm.go` para lógicas "custom" del ORM, invirtiendo las semánticas (el generador de ORM no usa la palabra ORM, pero el código manual sí).
- Un sufijo `_orm.go` actúa como un gran letrero de "Framework Territory". Disuade modificaciones manuales.
- El código escrito por el desarrollador para acceso a datos y reglas de negocio con inyección deberá seguir convenciones estándar como `repository.go` o `dao.go`.

---

## Stages

Para ejecutar este refactor, el agente (como Jules) deberá ejecutar los siguientes pasos de forma secuencial:

### Stage 1: Refactor Code Generator Output (`ormc`)
- Modificar la constante o variable en el código fuente de `cmd/ormc` (y en sus utilidades de generación) para que el sufijo de salida sea `_orm.go` en lugar de `_db.go`.
- Renombrar cualquier archivo `*_db.go` de los mocks o fixtures internos en `tinywasm/orm` que se utilice para tests a su nuevo equivalente `*_orm.go`.
- Modificar los tests unitarios e integrales de `ormc` que verifiquen o asuman la existencia de `_db.go` para que afirmen la existencia de `_orm.go`.

### Stage 2: Documentation Update
- Modificar `docs/SKILL.md`, reemplazando todas las referencias de archivos generados (como `model_db.go`) por `model_orm.go`.
- Actualizar el README principal (y cualquier documento en `docs/`) reflejando este cambio de convención para evitar confusiones a futuro.

### Stage 3: Ecosystem Test (Validation)
- Ejecutar `gotest` en la raíz de `tinywasm/orm` para asegurar que todo pase con el coverage esperado.
- Emitir la recomendación al usuario de iterar sobre los demás módulos (ej. `business-hours`, `appointment-booking`) para ejecutar un `ormc` fresco, borrar los obsoletos `model_db.go` y adaptar los planes/código al nuevo `model_orm.go` + `repository.go`.
