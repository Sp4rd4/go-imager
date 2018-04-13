CREATE TABLE IF NOT EXISTS "images" (
	"filename" varchar NOT NULL,
	"user_id" integer NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS "image_filename_idx" ON "images" ("filename");
CREATE INDEX IF NOT EXISTS "image_user_idx" ON "images" ("user_id");