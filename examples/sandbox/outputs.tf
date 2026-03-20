# Copyright 2026 Boozt Fashion AB
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0

output "ipam_url" {
  description = "IPAM Autopilot Cloud Run service URL. Pass as ipam_url variable on subsequent applies."
  value       = module.ipam.cloud_run_url
}

