package dcf_scoring

import data.classes
import input.class

has_key(x, k) { _ = x[k] }

weights[w] {
    has_key(classes,class)
    w:=classes[class]
}

weights[w] {
    not has_key(classes,class)
    w:=classes["default"]
}
