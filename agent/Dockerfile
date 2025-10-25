# syntax=docker/dockerfile:1.7
FROM node:20-bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive
SHELL ["/bin/bash", "-lc"]

# 1) База: пакеты для отладки и сети
# 2) Docker CLI + compose/buildx, НО без docker-ce (демона)
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates curl wget git gnupg ripgrep tini \
    # Network debugging
    iputils-ping traceroute netcat-openbsd nmap telnet dnsutils \
    iproute2 net-tools socat tcpdump \
    # Process/system debugging
    htop procps psmisc strace ltrace lsof \
    # File/text
    vim nano less tree file unzip zip jq \
    # Dev utils
    build-essential python3 python3-pip \
    # Monitoring
    iotop iftop nethogs \
    && rm -rf /var/lib/apt/lists/*

# Docker CLI репозиторий
RUN install -d -m 0755 /etc/apt/keyrings \
 && curl -fsSL https://download.docker.com/linux/debian/gpg \
    | gpg --dearmor -o /etc/apt/keyrings/docker.gpg \
 && echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/debian bookworm stable" \
    > /etc/apt/sources.list.d/docker.list \
 && apt-get update && apt-get install -y --no-install-recommends \
    docker-ce-cli docker-buildx-plugin docker-compose-plugin \
 && rm -rf /var/lib/apt/lists/*

# Claude Code в /usr/local/bin
RUN npm install -g @anthropic-ai/claude-code

# --- НЕРУТ ---
# Выстави нужный GID докер-сокета хоста, если хочешь доступ к /var/run/docker.sock
# На многих системах это 998 или 999. При необходимости переопредели на билде:
#   docker build --build-arg DOCKER_GID=999 -t image .
ARG USER=agent
ARG UID=998
ARG GID=998
ARG DOCKER_GID=999

RUN  groupadd -g "${GID}" "${USER}" \
 && useradd -m -u "${UID}" -g "${GID}" -s /bin/bash "${USER}" \
 && groupadd -g "${DOCKER_GID}" docker || true \
 && usermod -aG docker "${USER}"


ENV HOME=/home/${USER}
WORKDIR /home/${USER}

# Tini как PID 1
ENTRYPOINT ["tini","--"]
# Запуск под обычным пользователем
USER ${USER}

# По умолчанию запускаем claude в интерактивной оболочке
CMD ["bash","-lc","claude"]
