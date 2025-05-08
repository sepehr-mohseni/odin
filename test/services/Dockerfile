FROM node:18-alpine

WORKDIR /app

COPY package*.json ./
RUN npm install

COPY . .

EXPOSE 8081 8083 8084 8085

CMD ["npm", "run", "start:all"]
