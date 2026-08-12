# syntax=docker/dockerfile:1

# ---- build ----
FROM eclipse-temurin:25-jdk AS build
WORKDIR /src
# Resolve dependencies from the pom first so they cache across code changes.
COPY .mvn/ .mvn/
COPY mvnw pom.xml ./
RUN chmod +x mvnw && ./mvnw -B dependency:go-offline
COPY src/ src/
RUN ./mvnw -B clean package -DskipTests

# ---- runtime ----
# If eclipse-temurin:25-jre is not published for your platform, use 25-jdk here.
FROM eclipse-temurin:25-jre
WORKDIR /app
# Spring Boot repackaged jar (the plain one; *.jar.original is skipped).
COPY --from=build /src/target/*.jar /app/app.jar

# Run as non-root.
RUN useradd -r -u 1001 appuser
USER appuser

# SWIM / Solace credentials are NOT baked into the image. The app reads
# bridge/.env only as OPTIONAL config; in a container, supply the same values
# as environment variables (a k8s Secret) and Spring binds them identically.
ENTRYPOINT ["java", "-jar", "/app/app.jar"]
