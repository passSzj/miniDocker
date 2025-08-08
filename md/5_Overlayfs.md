OverlayFS 需要三个目录：

角色	含义
参数/目录	OverlayFS 术语	作用说明
lowerdir=b	只读层（lower）	提供基础内容，不能直接修改
upperdir=c	可写层（upper）	所有修改都发生在这一层
workdir=work	工作目录	OverlayFS 内部使用，必须提供
a	merge 目录	挂载点，展示 upper 和 lower 的合并视图

额外的 workdir 目录，用于 OverlayFS 的工作缓存。它必须和 upperdir 在同一个文件系统上。

读操作：
如果 c（upperdir）中有文件，优先使用 c 中的；
如果 c 中没有，才会去 b（lowerdir）中查找。

写操作：
如果你尝试修改 lowerdir 中的只读文件，OverlayFS 会自动将它从 b 拷贝到 c（copy-up），然后在 c 中修改；
删除操作是通过 whiteout 文件隐藏 lower 层的文件。