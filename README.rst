GoFlow Exporter
================

This application exports sFlow messages collected by `GoFlow <https://github.com/cloudflare/goflow>`_ to Prometheus.

How to use
-----------

1. Create a configuration file in JSON as below:

   ::

     {
       "brokers": [
         "127.0.0.1:9092"
       ],
       "topic": "flow-messages",
       "timeout": 10,
       "exporter_port": 30060
     }

#. Kick started the application:

   ::

     # export DEBUG=true
     ./goflow_exporter -c config.json
