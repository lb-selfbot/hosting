import os


for file in os.listdir("./vpn"):
    if not file.endswith(".ovpn"):
        continue

    with open(f"./vpn/{file}", "r") as f:
        content = f.read()

        content = content.replace(
            "auth-user-pass",
            "auth-user-pass auth.txt",
        )

    with open(f"./vpn/{file}", "w") as f:
        f.write(content)
