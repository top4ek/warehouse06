# Warehouse06

#### Run
Create `.env` file using .env_test as a template. Or just copy it.
Fire up `docker compose up` to start app and postgres servers:
```bash
docker compose up
```

Make some requests to create user record:
```bash
curl -H 'Content-Type: application/json' \
     -X POST \
     -d '{"user":{"name":"test","email":"test@example.com","password":"pass"}}' \
     localhost:3000/users
```
or to show all created records:
```bash
curl -H 'Content-Type: application/json' localhost:3000/users
```
or exact one:
```bash
curl -H 'Content-Type: application/json' localhost:3000/users/1
```
And system one feature, in praise of healthchecks:
```bash
curl -H 'Content-Type: application/json' localhost:3000/ping
```

#### TODO
- Add environments
- Add swagger
- Add documentation
- Add specs and Guard
- Add hot reloading in development env
- Move to falcon
- Add auth
- Harden validations
