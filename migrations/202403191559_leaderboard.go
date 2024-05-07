package migrations

import "app/lib"

var _ = lib.RegisterMigration("20240319155950_leaderboard", func(c *lib.Ctx) {
	c.DB.Execute(`
CREATE TABLE leaderboards_users (
  id text NOT NULL PRIMARY KEY,
  code text NOT NULL,
  referrer text NOT NULL,
  address text NOT NULL,
  social_id text NOT NULL,
  social_name text NOT NULL,
  social_username text NOT NULL,
  social_picture text NOT NULL,
  points bigint NOT NULL,
  points_referral bigint NOT NULL,
  created timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX leaderboards_users_code_idx ON leaderboards_users (code);
CREATE INDEX leaderboards_users_address_idx ON leaderboards_users (address);

CREATE TABLE leaderboards_points (
  id text NOT NULL PRIMARY KEY,
  user_id text NOT NULL REFERENCES leaderboards_users (id),
  reason text NOT NULL,
  reason_id text NOT NULL,
  points bigint NOT NULL,
  created timestamptz NOT NULL DEFAULT now()
);
`)
}, func(c *lib.Ctx) {
	c.DB.Execute(`
DROP TABLE leaderboards_points;
DROP TABLE leaderboards_users;
`)
})
