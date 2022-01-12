FROM maven
COPY src src
COPY pom.xml pom.xml

RUN mvn clean compile package

WORKDIR /
ENTRYPOINT ["java","-Dcom.sun.management.jmxremote","-Djava.rmi.server.hostname=localhost","-Dcom.sun.management.jmxremote.port=1088","-Dcom.sun.management.jmxremote.rmi.port=1088","-Dcom.sun.management.jmxremote.local.only=false","-Dcom.sun.management.jmxremote.authenticate=false","-Dcom.sun.management.jmxremote.ssl=false","-jar","./target/kafka-dummy-1.0-jar-with-dependencies.jar"]
