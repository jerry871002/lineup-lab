FROM node:20-alpine as builder

WORKDIR /app

ARG REACT_APP_STAT_API_BASE_URL=http://localhost:8082
ARG REACT_APP_SIMULATION_API_BASE_URL=http://localhost:8081

ENV REACT_APP_STAT_API_BASE_URL=$REACT_APP_STAT_API_BASE_URL
ENV REACT_APP_SIMULATION_API_BASE_URL=$REACT_APP_SIMULATION_API_BASE_URL

COPY package*.json ./
RUN npm ci

COPY . .

RUN npm run build

FROM nginx:1.27.0-alpine as runner

COPY --from=builder /app/build /usr/share/nginx/html

EXPOSE 80
