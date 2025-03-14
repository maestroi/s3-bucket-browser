# Build stage
FROM node:18-alpine AS build

WORKDIR /app

# Copy package.json and package-lock.json
COPY package*.json ./
RUN npm install

# Copy the rest of the code
COPY . .

# Build the app
RUN npm run build

# Production stage with Nginx
FROM nginx:alpine

# Copy the built app from the build stage
COPY --from=build /app/dist /usr/share/nginx/html

# Copy custom Nginx config
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"] 