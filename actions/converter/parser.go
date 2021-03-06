package converter

import (
	"reflect"
	"strings"
	"time"
	"regexp"
	c "simple-tool/common"
)

type (
	javaClassDescription struct {
		isPublic    bool
		className   string
		fields      []javaField
		parentClass *javaClassDescription
		innerClass  []javaClassDescription
		staticClass []javaClassDescription
	}
	javaFileDescription struct {
		packageName string
		classDesc   *javaClassDescription
		imports     []javaImport
		fileComment string
	}
	javaImport struct {
		importedClass string
	}
)

//基本go类型
const (
	GO_Number    = "float"
	GO_String    = "string"
	GO_Bool      = "bool"
	JAVA_Float   = "Float"
	JAVA_Integer = "Integer"
	JAVA_String  = "String"
	JAVA_Bool    = "Boolean"
	JAVA_Object  = "Object"
)

type javaField struct {
	filedName       string
	filedType       string
	fieldAnnotation fieldAnnotation
}

type fieldAnnotation struct {
	annotations []annotation
}

type annotation struct {
	name  string
	value string
}

var ImportConfig struct {
	hasJsonProperty bool
}

func JsonToJavaClass(name string, json map[string]interface{}) javaClassDescription {
	if json == nil {
		panic("param json should not be none")
	}

	classDesc := javaClassDescription{}
	classDesc.isPublic = true
	classDesc.className = name
	for _, key := range keys(json) {
		value := json[key]

		var annotation = fieldAnnotation{annotations: []annotation{}}
		processAnnotation(key, &annotation)

		key = c.LowerCaseFirst(key)

		//null,undefined 视为 Object
		if value == nil {
			var javaType = JAVA_Object
			classDesc.fields = append(classDesc.fields, javaField{key, javaType, annotation})
			continue
		}

		valueType := reflect.TypeOf(value)

		//基本类型 数字
		if strings.Contains(valueType.Name(), GO_Number) {
			var javaType string
			//判断是否是整数, fixme 0.00 也会识别为整数
			if float64(int(value.(float64)))-value.(float64) != 0 {
				javaType = JAVA_Float
			} else {
				javaType = JAVA_Integer
			}
			classDesc.fields = append(classDesc.fields, javaField{key, javaType, annotation})
		}
		//字符串
		if valueType.Name() == GO_String {
			classDesc.fields = append(classDesc.fields, javaField{key, JAVA_String, annotation})
		}
		//布尔值
		if valueType.Name() == GO_Bool {
			classDesc.fields = append(classDesc.fields, javaField{key, JAVA_Bool, annotation})
		}
		//对象
		if !strings.HasPrefix(valueType.String(), "[]") &&
			strings.Contains(valueType.String(), "map") {
			classDesc.fields = append(classDesc.fields,
				javaField{key, c.UpperCaseFirst(key), annotation})

			parentClass := findParent(&classDesc)
			objClass := JsonToJavaClass(c.UpperCaseFirst(key), value.(map[string]interface{}))
			objClass.parentClass = parentClass
			objClass.isPublic = false
			parentClass.innerClass = append(parentClass.innerClass, objClass)
		}

		//数组
		if strings.HasPrefix(valueType.String(), "[]") {
			if strings.HasPrefix(valueType.String(), "[][]") {
				panic("不支持数组嵌套")
			}
			nested := value.([]interface{})

			if len(nested) == 0 {
				var javaType = JAVA_Object
				classDesc.fields = append(classDesc.fields, javaField{key, javaType, annotation})
				continue
			}
			elm := nested[0]
			elmType := reflect.TypeOf(elm)
			//元素为对象
			if strings.Contains(elmType.String(), "map[string]") {

				classDesc.fields = append(classDesc.fields,
					javaField{key, listType(c.UpperCaseFirst(key)), annotation})

				parentClass := findParent(&classDesc)
				elmClass := JsonToJavaClass(c.UpperCaseFirst(key), elm.(map[string]interface{}))
				elmClass.parentClass = parentClass
				elmClass.isPublic = false
				parentClass.innerClass = append(parentClass.innerClass, elmClass)
				continue
			} else {
				//元素为基本值
				typeString := parseType(nested[0])
				classDesc.fields = append(classDesc.fields,
					javaField{key, listType(typeString), annotation})
			}
		}
	}
	return classDesc
}

func ClassToJavaFile(description *javaClassDescription) *javaFileDescription {
	javaFileDescription := javaFileDescription{}
	javaFileDescription.packageName = "generated"
	javaFileDescription.classDesc = description
	var imports []javaImport
	if ImportConfig.hasJsonProperty {
		imports = append(imports, javaImport{"com.fasterxml.jackson.annotation.JsonProperty"})
	}
	imports = append(imports, javaImport{"lombok.AllArgsConstructor"})
	imports = append(imports, javaImport{"lombok.Builder"})
	imports = append(imports, javaImport{"lombok.Data"})
	imports = append(imports, javaImport{"lombok.NoArgsConstructor"})
	if hasList(description) {
		imports = append(imports, javaImport{"java.util.List"})
	}

	javaFileDescription.imports = imports

	fileComment := `/**
* generated by simple-tool in $time
**/`
	timeString := time.Now().Format("2006/01/02 03:04:05")
	matcher, _ := regexp.Compile("\\$time")
	fileComment = matcher.ReplaceAllLiteralString(fileComment, timeString)
	javaFileDescription.fileComment = fileComment

	return &javaFileDescription
}

func processAnnotation(key string, annotationValue *fieldAnnotation) {
	if isUpperCaseFirst(key) {
		ImportConfig.hasJsonProperty = true
		annotationValue.annotations = append(annotationValue.annotations, annotation{"JsonProperty", key})
	}
}
