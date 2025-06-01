FROM node:18-alpine

WORKDIR /app

# Copy package.json and install dependencies
COPY package*.json ./
RUN npm install --only=production

# Copy all service files
COPY . .

# Expose common ports for test services
EXPOSE 8081 8082 8083 8084 8085

# Default command (can be overridden)
CMD ["npm", "start"]
