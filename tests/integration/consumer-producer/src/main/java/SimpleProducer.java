package kafkaDummy;

import java.util.Properties;
import org.apache.kafka.clients.producer.Producer;
import org.apache.kafka.clients.producer.KafkaProducer;
import org.apache.kafka.clients.producer.ProducerRecord;

public class SimpleProducer {
   public static void main(String[] args) {
      if(args.length < 3){
         System.out.println("Usage: producer <bootstrapBroker> <topic>");
         return;
      }

      String bootstrapBroker = args[1].toString();
      String topicName = args[2].toString();

      Properties props = new Properties();
      props.put("bootstrap.servers", bootstrapBroker);
      props.put("acks", "all");
      props.put("retries", 0);
      props.put("batch.size", 16384);
      props.put("linger.ms", 1);
      props.put("buffer.memory", 33554432);
      props.put("key.serializer",
         "org.apache.kafka.common.serialization.StringSerializer");
      props.put("value.serializer",
         "org.apache.kafka.common.serialization.StringSerializer");

      Producer<String, String> producer = new KafkaProducer
         <String, String>(props);

      int msg = 0;
      while(true) {
        producer.send(new ProducerRecord<String, String>(topicName,
        Integer.toString(msg), Integer.toString(msg)));
        System.out.println("Message sent successfully");
        msg++;
        wait(2000);
      }

      // never reached
      // producer.close();
   }

   public static void wait(int ms)
   {
       try
       {
           Thread.sleep(ms);
       }
       catch(InterruptedException ex)
       {
           Thread.currentThread().interrupt();
       }
   }
}