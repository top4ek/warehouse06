# Warehouse06

#### Run
Fire docker compose to start server:
```bash
docker compose up
```

Make some request to create:
```bash
curl -H 'Content-Type: application/json' \
     -X POST \
     -d '{"user":{"name":"test","email":"test@example.com","password":"pass"}}'
     localhost:3000/users
```
to show all created records:
```bash
curl -H 'Content-Type: application/json' localhost:3000/users
```
or exact one:
```bash
curl -H 'Content-Type: application/json' localhost:3000/users/1
```
And system one feature:
```bash
curl -H 'Content-Type: application/json' localhost:3000/ping
```

#### TODO
- Add environments
- Add specs and Guard
- Add swagger
