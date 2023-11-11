import os
import subprocess
from datetime import date

import boto3


def release():
    os.makedirs("build", exist_ok=True)
    for goos, arches in {
        "linux": ["arm64", "amd64", "386"],
        "darwin": ["arm64", "amd64"],
        "windows": ["amd64", "386"],
    }.items():
        for arch in arches:
            ext = ".exe" if goos == "windows" else ""
            version = date.today().isoformat().replace("-", "")
            filename = f"relational-sheets-{version}-{goos}-{arch}{ext}"
            subprocess.run(
                ["go", "build", "-o", f"build/{filename}"],
                env={
                    **os.environ,
                    "GOOS": goos,
                    "GOARCH": arch,
                }
            )
            publish(filename)


def publish(filename):
    s3 = boto3.client("s3")
    s3.upload_file(f"build/{filename}", "relational-sheets", filename)


if __name__ == "__main__":
    release()
