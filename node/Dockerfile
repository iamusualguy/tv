FROM jrottenberg/ffmpeg:4.1-alpine AS ffmpeg
FROM node:latest AS node
# COPY --from=ffmpeg / / 
COPY --from=node / / 
WORKDIR /usr/src/app
COPY index.js ./
# CMD [ "npm", "install" ]
CMD [ "node", "index.js" ]
