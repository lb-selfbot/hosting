FROM alpine:latest

# Create a new user
# RUN useradd -ms /bin/bash lightningbot
RUN adduser -D lightningbot

# Install and setup random VPN
RUN apk add --no-cache bash openvpn sudo
COPY ./vpn /etc/openvpn

# Run the bot
WORKDIR /home/lightningbot
COPY LB-Selfbot .
COPY start.sh .
CMD ["/bin/bash", "-c", "./start.sh"]
