This is two go services behind nginx reverse proxy:
* First is doing authentication via login/password and returning JWT
* Second is responsible for uploading and listing images for user

Authorization for images is done via user_id foreign key, user_id is passed around via JWT.
Middleware responsible for JWT checks is largely inspired by https://github.com/auth0/go-jwt-middleware

How to run
=====
`docker-compose up`

How to test
=====
`docker-compose -f docker-compose.test.yml run tests sh -c "./wait-for db:5432 -- go test ./... -count=1 -p 1 -v"`

`-p 1` parameter is responsible for running different packages tests serially, so they won't conflict on single db.