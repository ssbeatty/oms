package utils

import "strings"

var (
	folderIconMap = map[string]string{
		"dist":     "folder_dist",
		"doc":      "folder_docs",
		"docs":     "folder_docs",
		"git":      "folder_git",
		"download": "folder_download",
		"images":   "folder_images",
		"image":    "folder_images",
		"js":       "folder_javascript",
		"css":      "folder_css",
		"src":      "folder_src",
		"temp":     "folder_temp",
		"tmp":      "folder_temp",
		"test":     "folder_test",
		"vue":      "folder_vue",
		".git":     "folder_git",
	}
	extIconMap = map[string]string{
		// os
		"apk": "android",
		"exe": "exe",
		"db":  "database",
		"zip": "zip",
		"log": "log",
		"mod": "go-mod",

		// docs
		"doc":  "word",
		"docx": "word",
		"docm": "word",
		"md":   "markdown",
		"xls":  "xls",
		"xlsx": "xlsx",
		"csv":  "xls",
		"pdf":  "pdf",
		"ppt":  "ppt",
		"pptx": "ppt",

		// language
		"c":     "c",
		"cpp":   "cpp",
		"css":   "css",
		"go":    "go",
		"h":     "h",
		"html":  "html",
		"java":  "java",
		"js":    "javascript",
		"json":  "json",
		"kt":    "kotlin",
		"less":  "less",
		"lib":   "lib",
		"php":   "php",
		"py":    "python",
		"pyc":   "python",
		"pyd":   "python",
		"r":     "r",
		"rs":    "rust",
		"xml":   "xml",
		"yaml":  "yaml",
		"swift": "swift",
		"cs":    "csharp",
		"erl":   "erlang",

		// image
		"jpg": "image",
		"png": "image",
		"jar": "jar",
		"svg": "svg",

		// other
		"mp3": "mp3",
		"mp4": "mp4",
	}
	fileIconMap = map[string]string{
		"Dockerfile":     "docker",
		".eslintrc.js":   "eslint",
		"Jenkinsfile":    "jenkins",
		"nginx.conf":     "nginx",
		"yarn.lock":      "yarn",
		".gitignore":     "git",
		"CMakeLists.txt": "cmake",
		".gitlab-ci.yml": "gitlab",
		"Makefile":       "makefile",
		"robots.txt":     "robots",
		".vimrc":         "vim",
		".viminfo":       "vim",
	}
)

func GetFileExt(path string) string {
	args := strings.Split(path, ".")
	if len(args) < 2 {
		return ""
	} else if len(args) == 2 && args[0] == "" {
		return ""
	}

	// 特殊的后缀
	if strings.HasSuffix(path, ".tar.gz") {
		return "tar.gz"
	}
	return args[len(args)-1]
}

func GetFileIcon(fileName string, isDir bool) string {
	fileName = strings.ToLower(fileName)
	// 先判断目录
	if isDir {
		if icon, ok := folderIconMap[fileName]; ok {
			return icon
		}
		return ""
	} else {
		// 判断文件名
		if icon, ok := fileIconMap[fileName]; ok {
			return icon
		} else {
			// 否则后缀
			ext := GetFileExt(fileName)
			if ext == "" {
				return "file"
			}
			if icon, ok := extIconMap[ext]; ok {
				return icon
			}
			return ""
		}
	}
}
