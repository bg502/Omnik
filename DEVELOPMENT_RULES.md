# omnik Development Rules

## Critical Rule: Docker-First Development

### ⚠️ NEVER run commands outside Docker

**Always use:**
```bash
docker compose exec <service> <command>
docker compose run <service> <command>
docker compose build
docker compose up -d
```

**Never use:**
```bash
npm install          # ❌ Wrong
npm run dev          # ❌ Wrong
go build             # ❌ Wrong
python main.py       # ❌ Wrong
```

### Correct Workflow

#### Installing Dependencies
```bash
# ❌ Wrong
npm install

# ✅ Correct
docker compose build claude-bridge
docker compose up -d claude-bridge
```

#### Running Tests
```bash
# ❌ Wrong
npm test

# ✅ Correct
docker compose exec claude-bridge npm test
```

#### Development
```bash
# ❌ Wrong
npm run dev

# ✅ Correct
docker compose up -d claude-bridge
docker compose logs -f claude-bridge
```

#### Adding Tools/Dependencies
```bash
# ❌ Wrong - editing package.json and running npm install locally

# ✅ Correct
# 1. Edit package.json
# 2. Rebuild container
docker compose build claude-bridge
docker compose up -d claude-bridge
```

### Why This Matters

1. **Consistency** - Everyone uses same environment
2. **No local dependencies** - No need for Node.js/Go/Python on host
3. **Docker volumes** - Shared state (workspace, Claude auth)
4. **Production parity** - Dev matches production exactly

### Testing Checklist

- [ ] All changes tested inside Docker containers
- [ ] No local `node_modules` or `vendor` directories
- [ ] docker-compose.yml updated if services change
- [ ] Dockerfile updated if dependencies change
- [ ] Volumes properly mounted for shared state

## Service Communication

All inter-service communication goes through Docker network:

```
Go Bot → http://claude-bridge:9000  (✅ Docker network)
NOT
Go Bot → http://localhost:9000      (❌ Host network)
```

## Persistent Memory Notes

- Always use `docker compose` commands
- Test everything inside containers
- Document all Docker-specific configurations
- Keep track of volume mounts for shared state
