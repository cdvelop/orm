# Propuesta de Mejora de la API de tinywasm/orm

Este documento analiza el estado actual de la API de `tinywasm/orm`, identifica sus fortalezas y debilidades, y propone mejoras concretas para facilitar su adopción y autodocumentación, manteniendo las restricciones arquitectónicas (sin reflexión, compatible con WASM).

## 1. Análisis de la API Actual

### 1.1. Ventajas (Por qué es así)

La arquitectura actual está diseñada explícitamente para el entorno WebAssembly:

*   **Rendimiento y Tamaño del Binario (WASM Friendly):** Al evitar `reflect` y `database/sql`, el binario final es mucho más pequeño y rápido de iniciar, algo crítico en entornos WASM (navegador/edge).
*   **Agnosticismo Total:** La separación estricta entre `Model` (datos), `Query` (intención) y `Adapter` (ejecución) permite cambiar de PostgreSQL a IndexedDB o SQLite sin tocar una sola línea de lógica de negocio.
*   **Eficiencia de Memoria:** El patrón `ReadAll(factory, each)` permite procesar miles de registros sin forzar la creación de un slice gigante en memoria (streaming implícito).
*   **Seguridad de Tipos en Compilación:** Al forzar interfaces explícitas, muchos errores se detectan al compilar, no al ejecutar.

### 1.2. Desventajas y Puntos de Fricción

Sin embargo, esta rigidez introduce barreras de entrada para desarrolladores acostumbrados a ORMs más "mágicos" (GORM, TypeORM, Hibernate):

*   **Alto "Boilerplate":** Implementar `Values()` y `Pointers()` manualmente es tedioso y propenso a errores humanos (ej. cambiar el orden de un campo en `Columns` pero olvidar actualizar `Values`).
*   **"Magic Strings" (Cadenas Mágicas):** Las consultas dependen de cadenas literales para los nombres de columnas (`Eq("user_id", 1)`). Si el nombre de la columna cambia en la BBDD, el compilador no avisará del error en la query.
*   **Curva de Aprendizaje de `ReadAll`:** El patrón `factory/each` es extraño para quien solo quiere obtener un `[]User`. Requiere más código para el caso de uso más simple.
*   **Falta de Autodescubrimiento:** Un desarrollador nuevo no sabe qué columnas están disponibles para filtrar sin mirar la definición del struct o la base de datos.

---

## 2. Propuestas de Mejora

El objetivo es mejorar la **Developer Experience (DX)** sin sacrificar las restricciones técnicas (Zero Reflection).

### 2.1. Generación de Código (`tiny-gen`)

La solución más potente para eliminar el boilerplate sin usar reflexión es la generación de código.

**Propuesta:** Crear una herramienta CLI (`tiny-gen`) que lea los structs de Go y genere automáticamente los métodos de la interfaz `Model`.

**Antes (Manual):**
```go
type User struct { ID int; Name string }
func (u *User) Values() []any { return []any{u.ID, u.Name} } // Tedioso
```

**Después (Generado):**
El usuario solo define el struct y añade una anotación.
```go
//go:generate tiny-gen
type User struct {
    ID   int    `orm:"id,pk"`
    Name string `orm:"name"`
}
```
*Justificación:* Mantiene el runtime ligero (el código generado es código estático normal) pero elimina el error humano y el trabajo manual.

### 2.2. Tipado Fuerte para Columnas (Schema Definition)

Para eliminar los "Magic Strings", el generador de código debería crear constantes o estructuras que representen el esquema.

**Propuesta:**
Que `tiny-gen` genere un "Descriptor de Tabla".

```go
// Código Generado
var UserMeta = struct {
    TableName string
    ID        string
    Name      string
}{
    TableName: "users",
    ID:        "id",
    Name:      "name",
}
```

**Uso Mejorado:**
```go
// Autocompletado disponible! Si cambia el nombre, el código no compila.
db.Query(user).Where(orm.Eq(UserMeta.Name, "Alice"))
```
*Justificación:* Esto hace la API **autodocumentada**. El IDE sugiere los campos disponibles, reduciendo drásticamente la necesidad de consultar documentación externa o esquemas SQL.

### 2.3. Wrappers Genéricos (Go 1.18+)

Para facilitar la adopción, podemos ofrecer "sugar syntax" usando Generics para los casos comunes, manteniendo el `ReadAll` original para casos de alto rendimiento.

**Propuesta:** Añadir funciones auxiliares genéricas.

```go
// En el paquete orm
func Collect[T Model](qb *QB, factory func() T) ([]T, error) {
    var results []T
    err := qb.ReadAll(
        func() Model { return factory() },
        func(m Model) { results = append(results, m.(T)) },
    )
    return results, err
}
```

**Uso Mejorado:**
```go
users, err := orm.Collect(query, func() *User { return &User{} })
```
*Justificación:* Reduce la carga cognitiva para el 80% de los casos de uso (CRUD simple), acercando la experiencia a la de otros ORMs modernos.

### 2.4. Fluent API para Condiciones

Actualmente, construir condiciones complejas con `Or(Eq(...))` puede ser difícil de leer.

**Propuesta:** Mejorar el `QueryBuilder` para soportar construcción lógica más fluida.

```go
// Actual
qb.Where(orm.Or(orm.Eq("a", 1)))

// Propuesta (Fluent)
qb.Where("a").Eq(1).Or().Where("b").Gt(2)
```
*Justificación:* Esto es más legible y se lee como lenguaje natural. Sin embargo, esto requeriría cambios más profundos en la estructura interna de `QB` y `Condition`, por lo que se considera de menor prioridad que las propuestas 2.1 y 2.2.

---

## 3. Hoja de Ruta Sugerida

1.  **Fase 1 (Inmediata):** Implementar `Collect[T]` y helpers genéricos. Esto es un cambio puramente aditivo y de bajo coste.
2.  **Fase 2 (Herramientas):** Desarrollar `tiny-gen` para automatizar la implementación de `Model`. Esto es clave para la adopción masiva.
3.  **Fase 3 (Type Safety):** Extender `tiny-gen` para generar los metadatos de columnas (`UserMeta`) y eliminar los strings mágicos.

Estas mejoras transformarán `tinywasm/orm` de ser una "librería de bajo nivel eficiente" a un "framework productivo y seguro", sin violar sus principios fundacionales de rendimiento y compatibilidad WASM.
