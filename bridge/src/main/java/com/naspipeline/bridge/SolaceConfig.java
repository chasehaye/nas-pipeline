package com.naspipeline.bridge;

import jakarta.jms.ConnectionFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.jms.config.DefaultJmsListenerContainerFactory;
import org.springframework.jms.support.destination.JndiDestinationResolver;
import org.springframework.jndi.JndiTemplate;

import javax.naming.Context;
import java.util.Properties;

import org.springframework.jms.annotation.EnableJms;

@Configuration
@EnableJms

public class SolaceConfig {

    @Value("${solace.host}")
    private String host;

    @Value("${solace.vpn}")
    private String vpn;

    @Value("${solace.username}")
    private String username;

    @Value("${solace.password}")
    private String password;

    @Value("${solace.connection-factory}")
    private String connectionFactoryName;

    /**
     * JNDI context pointed at the Solace broker. Solace publishes the
     * connection factory (and often queues) through JNDI, so we look
     * them up by name rather than building them programmatically.
     */
    @Bean
    public JndiTemplate jndiTemplate() {
        Properties env = new Properties();
        env.put(Context.INITIAL_CONTEXT_FACTORY,
                "com.solacesystems.jndi.SolJNDIInitialContextFactory");
        env.put(Context.PROVIDER_URL, host);
        // Solace expects username@vpn as the security principal
        env.put(Context.SECURITY_PRINCIPAL, username + "@" + vpn);
        env.put(Context.SECURITY_CREDENTIALS, password);

        JndiTemplate template = new JndiTemplate();
        template.setEnvironment(env);
        return template;
    }

    @Bean
    public ConnectionFactory connectionFactory(JndiTemplate jndiTemplate) throws Exception {
        return jndiTemplate.lookup(connectionFactoryName, ConnectionFactory.class);
    }

    /**
     * Resolves destination names through JNDI as well. If your queue is
     * NOT published in JNDI, delete this bean and remove the
     * setDestinationResolver call below — Spring will then treat the
     * destination string as a physical queue name.
     */
    @Bean
    public JndiDestinationResolver destinationResolver(JndiTemplate jndiTemplate) {
        JndiDestinationResolver resolver = new JndiDestinationResolver();
        resolver.setJndiTemplate(jndiTemplate);
        // Don't blow up at startup if the name isn't in JNDI
        resolver.setFallbackToDynamicDestination(true);
        return resolver;
    }

    @Bean
    public DefaultJmsListenerContainerFactory jmsListenerContainerFactory(
            ConnectionFactory connectionFactory,
            JndiDestinationResolver destinationResolver) {

        DefaultJmsListenerContainerFactory factory =
                new DefaultJmsListenerContainerFactory();
        factory.setConnectionFactory(connectionFactory);
        factory.setDestinationResolver(destinationResolver);

        // Queue, not topic.
        factory.setPubSubDomain(false);

        // CLIENT_ACKNOWLEDGE (2): the message stays on the queue until we
        // explicitly acknowledge it. Once Kafka is in the picture, ack only
        // AFTER the Kafka write succeeds — otherwise a crash between the two
        // loses the message permanently.
        factory.setSessionAcknowledgeMode(2);

        // Single consumer while developing. Raising this parallelises
        // consumption but gives up per-queue ordering.
        factory.setConcurrency("1");

        return factory;
    }
}