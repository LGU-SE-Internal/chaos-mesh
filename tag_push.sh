#!/bin/bash

images=(
  "ghcr.io/chaos-mesh/chaos-mesh-e2e:latest"
  "ghcr.io/chaos-mesh/e2e-helper:latest"
  "ghcr.io/chaos-mesh/chaos-dashboard:latest"
  "ghcr.io/chaos-mesh/chaos-mesh:latest"
  "ghcr.io/chaos-mesh/chaos-daemon:latest"
)

target_registry="10.10.10.240/library"

for image in "${images[@]}"; do
  image_name=$(echo "$image" | awk -F'/' '{print $NF}')
  
  new_image="${target_registry}/${image_name}"

  echo "Tagging $image as $new_image"
  docker tag "$image" "$new_image"
  docker push "$new_image"
done