FROM node:20-alpine as builder

WORKDIR /app

ARG VITE_STAT_API_BASE_URL
ARG VITE_SIMULATION_API_BASE_URL

ENV VITE_STAT_API_BASE_URL=$VITE_STAT_API_BASE_URL
ENV VITE_SIMULATION_API_BASE_URL=$VITE_SIMULATION_API_BASE_URL

COPY package*.json ./
RUN npm ci

COPY . .

RUN npm run build

FROM nginx:1.27.0-alpine as runner

COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80
