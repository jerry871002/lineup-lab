FROM postgres:16.4

COPY ./init-table.sql ./seed-demo.sql /docker-entrypoint-initdb.d/

EXPOSE 5432
