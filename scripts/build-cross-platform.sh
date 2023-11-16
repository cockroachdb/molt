#!/bin/bash

package=$1
original_package_output_name=$2
version=$3
if [[ -z "$package" || -z "package_name" ]]; then
  echo "usage: $0 <package-name> <package-output-name> <version>"
  exit 1
fi
	
platforms=("windows/amd64" "windows/arm64" "darwin/arm64" "darwin/amd64" "linux/amd64" "linux/arm64")

for platform in "${platforms[@]}"
do
	platform_split=(${platform//\// })
	GOOS=${platform_split[0]}
	GOARCH=${platform_split[1]}
	output_name=$original_package_output_name'-'$version'.'$GOOS'-'$GOARCH'.tgz'
	latest_output_name=$original_package_output_name'-latest.'$GOOS'-'$GOARCH'.tgz'
	if [ $GOOS = "windows" ]; then
		package_output_name=$original_package_output_name'.exe'
	else
		package_output_name=$original_package_output_name
	fi	

	# Build for the given OS and architecture.
	echo "Building for $platform."
	env GOOS=$GOOS GOARCH=$GOARCH go build -o $package_output_name $package
	if [ $? -ne 0 ]; then
   		echo "Failed to build for $platform"
		exit 1
	fi

	# Compress for the given build package.
	original_path=$(pwd)
	cd $(dirname $package_output_name)
	tar -cvzf $(basename $output_name) $(basename $package_output_name)
	tar -cvzf $(basename $latest_output_name) $(basename $package_output_name)
	rm $(basename $package_output_name)

	echo "Finished building for $output_name."
	cd $original_path
done