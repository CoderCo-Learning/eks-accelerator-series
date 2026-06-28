terraform {
  required_version = ">= 1.7"

  required_providers {
    aws = {
      source = "hashicorp/aws"
      # Pin the major version. `~> 6.0` allows 6.x but never jumps to 7.0, where
      # the breaking changes land. Keep a lock file so the exact build is reproducible.
      version = "~> 6.0"
    }
  }
}
