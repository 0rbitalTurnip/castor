# Build from the repository root: docker build -f app/Dockerfile .
FROM node:24-bookworm-slim AS build
WORKDIR /src
ENV NEXT_TELEMETRY_DISABLED=1
COPY package.json yarn.lock .yarnrc.yml ./
COPY .yarn/releases/ .yarn/releases/
COPY app/package.json app/package.json
RUN node .yarn/releases/yarn-4.18.1.cjs install --immutable
COPY app/ app/
RUN node .yarn/releases/yarn-4.18.1.cjs workspace castor-app typecheck \
 && node .yarn/releases/yarn-4.18.1.cjs workspace castor-app test \
 && node .yarn/releases/yarn-4.18.1.cjs workspace castor-app build

FROM node:24-bookworm-slim AS runtime
WORKDIR /app
ENV NODE_ENV=production NEXT_TELEMETRY_DISABLED=1 PORT=3000 HOSTNAME=127.0.0.1
COPY --from=build --chown=node:node /src/app/.next/standalone/ ./
COPY --from=build --chown=node:node /src/app/.next/static/ ./app/.next/static/
USER node
EXPOSE 3000
HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
  CMD node -e "fetch('http://'+(process.env.HOSTNAME==='0.0.0.0'?'127.0.0.1':process.env.HOSTNAME)+':'+process.env.PORT+'/healthz',{signal:AbortSignal.timeout(4000)}).then(r=>process.exit(r.ok?0:1)).catch(()=>process.exit(1))"
CMD ["node", "app/server.js"]
