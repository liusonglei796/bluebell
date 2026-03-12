# GORM 数据存取与类型映射核心指南

## 1. GORM 的安全存取原理

- **如何“安全存入”**：GORM 底层强制使用**预编译指令 (Prepared Statement)** 和**参数化查询 (`?` 占位符)**，用户输入的内容统统作为参数传递给数据库，从根本上杜绝 SQL 注入。
- **如何“完整读取”**：根据在内存中缓存的映射模型 Schema，利用 Go 的**反射（Reflect）**机制在运行时动态实例化结构体，并根据列名精准赋值给结构体字段。

---

## 2. 为什么本项目完全不需要写 Scanner / Valuer？

本项目的模型设计（如 `User` 和 `Post`）完美契合了关系型数据库的扁平化规范，全军覆没只用了**底层驱动原生支持**的类型：

1. **基础类型免配置**：`int64`、`int`、`string` 等纯粹的数据类型，Go 底层的 `database/sql` 驱动已经实现了完美的自动双向转换。
2. **时间类型自带支持**：结构体嵌套了 `gorm.Model`，内部使用的 `time.Time` 以及 `gorm.DeletedAt` 类型，Go 标准库与 GORM 官方已经默默帮你实现了 `Scanner` 和 `Valuer` 接口。
3. **关联对象无需单列存储**：例如 `Post` 模型里的 `Author *User`。它并不是作为一个 JSON 或二级制大对象存放在 `post` 表的一个列里，而是通过存 `AuthorID` 外键，查询时配合 `db.Preload("Author")` 发起两次 SQL 查询后自动在内存中完成组装装配。

---

## 3. 什么时候才“必须”手写 Scanner / Valuer？

只有当你**想把 Go 的复杂类型（如切片、字典或自定义嵌套结构体）硬塞进数据库的“一个单列”中时**，底层驱动不知道该怎么转换它，才必须通过这两个接口来写“翻译规则”。

**示例：强行将 `[]string` 标签列表存进数据库的 `tags` 列（以 JSON 形式）**

```go
import (
	"database/sql/driver"
	"encoding/json"
)

// 1. 声明一个自定义类型
type StringSlice []string

// 2. 实现 Valuer (存入时触发)：转为 JSON 字节交给数据库
func (s StringSlice) Value() (driver.Value, error) {
	return json.Marshal(s)
}

// 3. 实现 Scanner (读出时触发)：把数据库返回的 JSON 解码还原为切片
func (s *StringSlice) Scan(value interface{}) error {
	b, _ := value.([]byte)
	return json.Unmarshal(b, s)
}

// 在模型中使用
type Post struct {
	// ... 
	Tags StringSlice `gorm:"column:tags;type:json"` 
}
```
