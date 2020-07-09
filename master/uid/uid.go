package uid

import (
	"github.com/rs/xid"
	"github.com/mobmob912/takuhai/master/master/repository"
)

type uid struct{}

func NewUIDGenerator() repository.UID {
	return &uid{}
}

func (u *uid) New() string {
	return xid.New().String()
}
