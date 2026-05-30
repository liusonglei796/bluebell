package di

import (
	"bluebell/internal/application"
	postsvc "bluebell/internal/application/post"
	socialsvc "bluebell/internal/application/social"
	usersvc "bluebell/internal/application/user"
	"bluebell/internal/config"
	"bluebell/internal/domain"
	"bluebell/internal/infrastructure/mq"
	mysqlrepo "bluebell/internal/infrastructure/persistence/mysql"
	redisrepo "bluebell/internal/infrastructure/persistence/redis"
)

type Services struct {
	Post      application.PostService
	Community *application.CommunityService
	User      application.UserService
	Social    application.SocialService
	Bookmark  *application.BookmarkService
}

func NewServices(
	dbRepos *mysqlrepo.Repositories,
	cacheRepos *redisrepo.Repositories,
	tokenService domain.TokenService,
	searchRepo domain.PostSearchRepository,
	searchSyncRepo domain.PostSearchSyncRepository,
	publisher *mq.Publisher,
	cfg *config.Config,
) *Services {
	return &Services{
		Post:      postsvc.NewPostService(dbRepos.Post, cacheRepos.PostCache, dbRepos.Vote, dbRepos.Remark, publisher, searchRepo, searchSyncRepo),
		Community: application.NewCommunityService(dbRepos.Community, dbRepos.User),
		User:      usersvc.NewUserService(dbRepos.User, dbRepos.Social, cacheRepos.TokenCache, tokenService),
		Social:    socialsvc.NewSocialService(dbRepos.Social, dbRepos.User, publisher),
		Bookmark:  application.NewBookmarkService(dbRepos.Bookmark, dbRepos.Post, dbRepos.User, dbRepos.Community),
	}
}