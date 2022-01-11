package kafkaDummy;

import java.util.Properties;
import java.util.Arrays;
import org.apache.kafka.clients.consumer.KafkaConsumer;
import org.apache.kafka.clients.consumer.ConsumerRecords;
import org.apache.kafka.clients.consumer.ConsumerRecord;

public class SimpleConsumer {
   public static void main(String[] args) {
      if(args.length < 4){
         System.out.println("Usage: consumer <bootstrapBroker> <topic> <groupname>");
         return;
      }

      String bootstrapBroker = args[1].toString();
      String topic = args[2].toString();
      String group = args[3].toString();

      Properties props = new Properties();
      props.put("bootstrap.servers", bootstrapBroker);
      props.put("group.id", group);
      props.put("enable.auto.commit", "true");
      props.put("auto.commit.interval.ms", "1000");
      props.put("session.timeout.ms", "30000");
      props.put("key.deserializer",
         "org.apache.kafka.common.serialization.StringDeserializer");
      props.put("value.deserializer",
         "org.apache.kafka.common.serialization.StringDeserializer");

      KafkaConsumer<String, String> consumer = new KafkaConsumer<String, String>(props);

      consumer.subscribe(Arrays.asList(topic));
      System.out.println("Subscribed to topic " + topic);

      while (true) {
         ConsumerRecords<String, String> records = consumer.poll(10);
         for (ConsumerRecord<String, String> record : records) {
            System.out.printf("offset = %d, key = %s, value = %s\n", record.offset(), record.key(), record.value());
         }
      }
   }
}