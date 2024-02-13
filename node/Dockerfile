FROM node:latest AS node
WORKDIR /usr/src/app
RUN mkdir static
COPY package.json ./
RUN npm install
RUN apt update && apt install -y ffmpeg && apt install -y imagemagick
COPY . .
VOLUME /usr/src/app/video
EXPOSE 3001
CMD [ "node", "index.js" ]
