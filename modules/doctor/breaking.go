// Copyright 2022 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package doctor

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/log"

	"xorm.io/builder"
)

func iterateUserAccounts(ctx context.Context, each func(*user.User) error) error {
	err := db.Iterate(
		ctx,
		new(user.User),
		builder.Gt{"id": 0},
		func(idx int, bean interface{}) error {
			return each(bean.(*user.User))
		},
	)
	return err
}

// Since 1.16.4 new restrictions has been set on email addresses. However users with invalid email
// addresses would be currently facing a error due to their invalid email address.
// Ref: https://github.com/go-gitea/gitea/pull/19085 & https://github.com/go-gitea/gitea/pull/17688
func checkUserEmail(ctx context.Context, logger log.Logger, _ bool) error {
	// We could use quirky SQL to get all users that start without a [a-zA-Z0-9], but that would mean
	// DB provider-specific SQL and only works _now_. So instead we iterate trough all user accounts and
	// use the user.ValidateEmail function to be future-proof.
	var invalidUserCount int64
	if err := iterateUserAccounts(ctx, func(u *user.User) error {
		// Only check for users, skip
		if u.Type != user.UserTypeIndividual {
			return nil
		}

		if err := user.ValidateEmail(u.Email); err != nil {
			invalidUserCount++
			logger.Warn("User[id=%d name=%q] have not a valid e-mail: %v", u.ID, u.Name, err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("iterateUserAccounts: %v", err)
	}

	if invalidUserCount == 0 {
		logger.Info("All users have a valid e-mail.")
	} else {
		logger.Warn("%d user(s) have a non-valid e-mail.", invalidUserCount)
	}
	return nil
}

func init() {
	Register(&Check{
		Title:     "Check if users has an valid email address",
		Name:      "check-user-email",
		IsDefault: false,
		Run:       checkUserEmail,
		Priority:  9,
	})
}
