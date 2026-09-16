# =============================================================
# Frontend — omie-sync (Vue 3 + Vite)
# Multi-stage: build com Node 20, runtime com nginx Alpine
# =============================================================

FROM node:20-alpine AS builder

WORKDIR /app

COPY package*.json ./
RUN npm ci --prefer-offline

COPY . .

# VITE_API_URL=/api faz o frontend chamar rotas relativas,
# que o nginx redireciona para o backend via rede interna.
ENV VITE_API_URL=/api

RUN npm run build

# -------------------------------------------------------------

FROM nginx:alpine

COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf.template /etc/nginx/templates/default.conf.template

# BACKEND_HOST é o apelido de rede Docker (Coolify "Network Aliases") do
# backend deste MESMO ambiente. O valor default aqui preserva o
# comportamento de antes desta variável existir (HML usa o alias "backend").
# Cada ambiente precisa do seu próprio par (alias único no backend +
# BACKEND_HOST correspondente no frontend) — dois ambientes que
# compartilham a mesma rede Docker e o mesmo alias fazem o frontend de um
# conversar com o backend do outro sem erro nenhum visível.
ENV BACKEND_HOST=backend:8080

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
