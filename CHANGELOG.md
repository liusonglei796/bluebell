# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-02-19

### Features
- 添加帖子软删除功能
- 切换到多阶段 Docker 构建
- 添加速率限制和超时中间件
- 重构 DTOs 并添加 Docker 支持（含 Windows 辅助脚本）

### Refactors
- 使用 ZRangeArgsWithScores 优化投票数据查询
- 移动帖子列表调度器到 controller
- 优化 controller/post.go 代码和注释
- 优化 Post 模块代码结构和命名
- 优化 RefreshToken 逻辑，移除冗余数据库查询
- 将密码加密逻辑移动到 Model 的 BeforeCreate 钩子
- 标准化 User 和 Community 信息对象命名
- 从 sqlx 迁移到 GORM ORM 框架
- 重构错误处理架构，引入 errorx 包实现三层错误分离

### Documentation
- 重构压力测试指南，包含完整场景

### Chores
- 删除不再需要的配置文件
- 更新 go.mod 依赖
