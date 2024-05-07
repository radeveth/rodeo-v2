package migrations

import "app/lib"

var _ = lib.RegisterMigration("202001010000_initial", func(c *lib.Ctx) {
	c.DB.Execute(`
CREATE TABLE users (
  id text NOT NULL PRIMARY KEY,
  name text NOT NULL,
  email text NOT NULL,
  password text NOT NULL,
  created timestamptz NOT NULL DEFAULT now(),
  updated timestamptz NOT NULL DEFAULT now(),
  deleted timestamptz
);
CREATE INDEX users_email_idx ON users (email);

CREATE TABLE sessions (
  id text NOT NULL PRIMARY KEY,
  user_id text NOT NULL REFERENCES users (id),
  data jsonb NOT NULL,
  expires timestamptz NOT NULL DEFAULT now(),
  created timestamptz NOT NULL DEFAULT now(),
  updated timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX sessions_user_id_idx ON sessions (user_id);

CREATE TABLE posts (
  id text NOT NULL PRIMARY KEY,
  slug text NOT NULL,
  title text NOT NULL,
  content text NOT NULL,
  published bool NOT NULL,
  created timestamptz NOT NULL DEFAULT now(),
  updated timestamptz NOT NULL DEFAULT now(),
  deleted timestamptz
);

CREATE TABLE docs (
  id text NOT NULL PRIMARY KEY,
  slug text NOT NULL,
  title text NOT NULL,
  content text NOT NULL,
  published bool NOT NULL,
  created timestamptz NOT NULL DEFAULT now(),
  updated timestamptz NOT NULL DEFAULT now(),
  deleted timestamptz
);

CREATE TABLE positions (
  id bigint NOT NULL PRIMARY KEY,
  chain int NOT NULL,
  "index" int NOT NULL,
  pool text NOT NULL,
  strategy text NOT NULL,
  shares decimal NOT NULL,
  borrow decimal NOT NULL,
  shares_value decimal NOT NULL,
  borrow_value decimal NOT NULL,
  life decimal NOT NULL,
  amount decimal NOT NULL,
  price decimal NOT NULL,
  created timestamptz NOT NULL DEFAULT now(),
  updated timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX positions_idx ON positions (chain, "index");
  `)
}, func(c *lib.Ctx) {
	c.DB.Execute(`
DROP TABLE positions;
DROP TABLE docs;
DROP TABLE posts;
DROP TABLE sessions;
DROP TABLE users;
  `)
})
