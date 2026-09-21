package mysql

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"bluebell/internal/model"

	"gorm.io/gorm"
)

// CommunityDao 社区数据访问对象 (集成 5 分钟本地内存缓存)
type CommunityDao struct {
	db        *gorm.DB
	mu        sync.RWMutex
	listCache []*model.Community
	mapCache  map[int64]*model.Community
	expireAt  time.Time
}

// NewCommunityDao 创建社区 DAO 实例
func NewCommunityDao(db *gorm.DB) *CommunityDao {
	return &CommunityDao{
		db:       db,
		mapCache: make(map[int64]*model.Community),
	}
}

// GetCommunityList 查询社区列表数据（优先使用本地内存缓存，0 SQL）
func (d *CommunityDao) GetCommunityList(ctx context.Context) (data []*model.Community, err error) {
	d.mu.RLock()
	if time.Now().Before(d.expireAt) && d.listCache != nil {
		cached := make([]*model.Community, len(d.listCache))
		copy(cached, d.listCache)
		d.mu.RUnlock()
		return cached, nil
	}
	d.mu.RUnlock()

	d.mu.Lock()
	defer d.mu.Unlock()
	// 双重检查锁定 (DCL)
	if time.Now().Before(d.expireAt) && d.listCache != nil {
		cached := make([]*model.Community, len(d.listCache))
		copy(cached, d.listCache)
		return cached, nil
	}

	err = d.db.WithContext(ctx).Select("id", "community_name", "introduction").Find(&data).Error
	if err != nil {
		return nil, fmt.Errorf("查询社区列表失败: %w", err)
	}

	d.listCache = data
	d.mapCache = make(map[int64]*model.Community, len(data))
	for _, c := range data {
		d.mapCache[int64(c.ID)] = c
	}
	d.expireAt = time.Now().Add(5 * time.Minute)

	cached := make([]*model.Community, len(data))
	copy(cached, data)
	return cached, nil
}

// GetCommunityDetailByID 根据ID查询社区详情（优先使用本地内存缓存，0 SQL）
func (d *CommunityDao) GetCommunityDetailByID(ctx context.Context, id int64) (*model.Community, error) {
	d.mu.RLock()
	if time.Now().Before(d.expireAt) && d.mapCache != nil {
		if c, ok := d.mapCache[id]; ok {
			d.mu.RUnlock()
			return c, nil
		}
	}
	d.mu.RUnlock()

	// 未命中或者已过期，回源数据库
	m := new(model.Community)
	err := d.db.WithContext(ctx).Where("id = ?", id).First(m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询社区详情失败: %w", err)
	}

	d.mu.Lock()
	if d.mapCache == nil {
		d.mapCache = make(map[int64]*model.Community)
	}
	d.mapCache[id] = m
	d.mu.Unlock()

	return m, nil
}

// CreateCommunity 创建新社区（创建成功后使本地缓存失效）
// 并发创建同名社区时，唯一索引会让数据库直接返回 1062 重复键错误（TranslateError 下为 gorm.ErrDuplicatedKey），
// 这里将其映射为 ErrDuplicate，由 controller 映射为 409。单条 INSERT 由数据库兜底，不存在 check-then-insert 的竞态窗口。
func (d *CommunityDao) CreateCommunity(ctx context.Context, community *model.Community) error {
	err := d.db.WithContext(ctx).Create(community).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return model.ErrDuplicate
		}
		return fmt.Errorf("创建社区失败: %w", err)
	}

	// 失效本地缓存
	d.mu.Lock()
	d.listCache = nil
	d.expireAt = time.Time{}
	d.mu.Unlock()

	return nil
}
