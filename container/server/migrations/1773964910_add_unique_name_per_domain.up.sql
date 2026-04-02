ALTER TABLE subnets ADD UNIQUE KEY uq_name_routing_domain (name, routing_domain_id);
