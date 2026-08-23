FROM node:22.23.1-alpine3.24@sha256:16e22a550f3863206a3f701448c45f7912c6896a62de43add43bb9c86130c3e2 AS builder

WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
ARG VITE_PUBLIC_BASE_URL
ENV VITE_PUBLIC_BASE_URL=$VITE_PUBLIC_BASE_URL
ENV VITE_API_BASE_URL=
RUN npm run build

FROM nginx:1.30.4-alpine3.24@sha256:97d490c12ba55b4946b01546d1c3ed324e8d41ab1c9fcb2a616aa470620e5b46
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY security-headers.conf /etc/nginx/security-headers.conf
COPY --from=builder /app/dist /usr/share/nginx/html
EXPOSE 8080
