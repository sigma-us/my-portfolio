variable "versioning" {
    description = "Enable versioning on s3 bucket? (Enabled/Disabled)"
    type = string
    default = "Disabled"
}

variable "acl" {
    description = "S3 Access Control List (private/public/public-read)"
    default = "private"
}


variable bucket_name {
    description = "Name for bucket"
    type = string
}


variable block_public {
    description = "Block all public Access"
    type = bool
    default = true
}
