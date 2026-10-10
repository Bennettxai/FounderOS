# Lightweight workspace container for BusinessOS
FROM alpine:3.19

# Install essential tools in a single layer to keep image small
RUN apk add --no-cache \
    bash \
    zsh \
    git \
    curl \
    jq \
    vim \
    nano \
    python3 \
    py3-pip \
    nodejs \
    npm \
    ca-certificates \
    tzdata \
    && rm -rf /var/cache/apk/*

# Create non-root user with UID 1000
RUN addgroup -g 1000 workspace && \
    adduser -D -u 1000 -G workspace -s /bin/bash workspace

# Set working directory
WORKDIR /workspace

# Change ownership of workspace directory
RUN chown -R workspace:workspace /workspace

# Switch to non-root user
USER workspace

# Set default shell
ENV SHELL=/bin/bash

# Copy BusinessOS init script
COPY --chown=workspace:workspace businessos_init.sh /home/workspace/.businessos_init.sh
RUN chmod +x /home/workspace/.businessos_init.sh

# Add common aliases and environment setup
RUN echo 'alias ll="ls -lah"' >> /home/workspace/.bashrc && \
    echo 'alias la="ls -A"' >> /home/workspace/.bashrc && \
    echo 'alias l="ls -CF"' >> /home/workspace/.bashrc && \
    echo 'export PS1="\[\033[01;32m\]\u@\h\[\033[00m\]:\[\033[01;34m\]\w\[\033[00m\]\$ "' >> /home/workspace/.bashrc && \
    echo 'export BUSINESSOS_API_URL="http://host.docker.internal:8001"' >> /home/workspace/.bashrc && \
    echo '[ -f ~/.businessos_init.sh ] && source ~/.businessos_init.sh' >> /home/workspace/.bashrc

# Set up zsh config
RUN echo 'alias ll="ls -lah"' >> /home/workspace/.zshrc && \
    echo 'alias la="ls -A"' >> /home/workspace/.zshrc && \
    echo 'alias l="ls -CF"' >> /home/workspace/.zshrc && \
    echo 'export PS1="%F{green}%n@%m%f:%F{blue}%~%f$ "' >> /home/workspace/.zshrc && \
    echo 'export BUSINESSOS_API_URL="http://host.docker.internal:8001"' >> /home/workspace/.zshrc && \
    echo '[ -f ~/.businessos_init.sh ] && source ~/.businessos_init.sh' >> /home/workspace/.zshrc

# Default command
CMD ["/bin/bash"]
