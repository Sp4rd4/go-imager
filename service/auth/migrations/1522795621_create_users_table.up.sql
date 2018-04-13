CREATE TABLE IF NOT EXISTS "users" (
	"id" BIGSERIAL PRIMARY KEY NOT NULL,
	"login" varchar NOT NULL,
	"password_hash" varchar NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS "user_login_idx" ON "users" ("login");