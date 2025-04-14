#!/bin/bash

# 原始镜像列表
images=(
  "ghcr.io/chaos-mesh/chaos-mesh-e2e:latest"
  "ghcr.io/chaos-mesh/e2e-helper:latest"
  "ghcr.io/chaos-mesh/chaos-dashboard:latest"
  "ghcr.io/chaos-mesh/chaos-mesh:latest"
  "ghcr.io/chaos-mesh/chaos-daemon:latest"
)

# 目标仓库前缀
target_registry="10.10.10.240/library"

# 循环 tag 镜像
for image in "${images[@]}"; do
  # 提取镜像名（去掉 ghcr.io/chaos-mesh/）
  image_name=$(echo "$image" | awk -F'/' '{print $NF}')
  
  # 构建新镜像名
  new_image="${target_registry}/${image_name}"

  echo "Tagging $image as $new_image"
  docker tag "$image" "$new_image"
  docker push "$new_image"
done