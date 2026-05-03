FROM node:22-bookworm AS test

WORKDIR /app

COPY public/package*.json ./
RUN npm ci || npm install

COPY public/ ./

RUN npm run test:unit 2>&1 | tee frontend-unit.log
RUN touch /test-passed

FROM nginx:1.27-alpine

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY public/ /usr/share/nginx/html/

COPY --from=test /test-passed /tmp/test-passed
COPY --from=test /app/frontend-unit.log /frontend-unit.log

EXPOSE 80