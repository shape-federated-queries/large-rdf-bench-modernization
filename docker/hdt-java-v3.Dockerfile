# hdt-java v3 with -cattree/-disk bounded-memory HDT generation. The official
# rdfhdt/hdt-java image is a 2018 build (v2.1, in-memory only), so we build v3
# ourselves. Needed for the datasets whose dictionary exceeds RAM.
FROM maven:3.9-eclipse-temurin-11 AS build
ARG HDT_VERSION=3.0.10
ADD https://github.com/rdfhdt/hdt-java/archive/refs/tags/v${HDT_VERSION}.tar.gz /src.tar.gz
RUN tar xzf /src.tar.gz -C / && mv /hdt-java-${HDT_VERSION} /src
WORKDIR /src
RUN mvn -q -B -DskipTests clean install
RUN d="$(dirname "$(find hdt-java-package/target -type f -name rdf2hdt.sh | head -1)")" \
	&& cp -r "$d/.." /opt/hdt-java

FROM eclipse-temurin:11-jre
COPY --from=build /opt/hdt-java /opt/hdt-java
ENV PATH="/opt/hdt-java/bin:${PATH}"
