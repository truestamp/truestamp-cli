# Copyright © 2020-2022 Truestamp Inc. All rights reserved.

# These Make commands are called from 'deno task' commands. They are only here since Deno does not support some of the built in commands.
# See : https://deno.land/manual@v1.25.2/tools/task_runner

compress-darwin:
	cd ./build && for i in truestamp-darwin*; do mv $$i truestamp && tar -czf $$i.tar.gz truestamp && rm truestamp; done && cd -

compress-linux:
	cd ./build && for i in truestamp-linux*; do mv $$i truestamp && tar -czf $$i.tar.gz truestamp && rm truestamp; done && cd -

compress-windows:
	cd ./build && for i in truestamp-windows*; do mv $$i truestamp.exe && zip -r truestamp-windows-x86_64.zip truestamp.exe && rm truestamp.exe; done && cd -
