package kafkaDummy;

public class Factory {
    public static void main(String[] args) {
        if (args[0].equals("producer")) {
            SimpleProducer.main(args);
        } else if (args[0].equals("consumer")) {
            SimpleConsumer.main(args);
        } else {
            System.out.println("First argument should be consumer/producer" + args[0]);
        }
    }
}