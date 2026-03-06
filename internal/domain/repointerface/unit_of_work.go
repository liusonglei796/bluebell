package repointerface

// UnitOfWork 工作单元接口
// 提供事务支持和 Repository 访问器
// Service 层通过此接口访问多个 Repository 并在事务中协调操作
type UnitOfWork interface {
	PostRepo() PostRepository
	CommunityRepo() CommunityRepository
	UserRepo() UserRepository
	// Transaction 在数据库事务中执行函数
	// 回调参数 uow 是绑定了事务连接的新 UnitOfWork 实例
	Transaction(fn func(uow UnitOfWork) error) error
}
