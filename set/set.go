package set

// Set represents a set of unique elements
type Set[T comparable] map[T]struct{}

// Of creates a new set with the given elements
func Of[T comparable](elements ...T) Set[T] {
    s := make(Set[T], len(elements))
    for _, e := range elements {
        s[e] = struct{}{}
    }
    return s
}

// Add adds an element to the set
func (s Set[T]) Add(element T) {
    s[element] = struct{}{}
}

// Contains checks if an element is in the set
func (s Set[T]) Contains(element T) bool {
    _, exists := s[element]
    return exists
}

// Remove removes an element from the set
func (s Set[T]) Remove(element T) {
    delete(s, element)
}

// Len returns the number of elements in the set
func (s Set[T]) Len() int {
    return len(s)
}

// List returns a slice of all elements in the set
func (s Set[T]) List() []T {
    result := make([]T, 0, len(s))
    for element := range s {
        result = append(result, element)
    }
    return result
}

// Clear removes all elements from the set
func (s Set[T]) Clear() {
    for k := range s {
        delete(s, k)
    }
}

// Union returns a new set containing all elements from both sets
func (s Set[T]) Union(other Set[T]) Set[T] {
    result := make(Set[T])
    for k := range s {
        result[k] = struct{}{}
    }
    for k := range other {
        result[k] = struct{}{}
    }
    return result
}

// Intersection returns a new set containing only elements in both sets
func (s Set[T]) Intersection(other Set[T]) Set[T] {
    result := make(Set[T])
    for k := range s {
        if other.Contains(k) {
            result[k] = struct{}{}
        }
    }
    return result
}

// Difference returns a new set containing elements in s but not in other
func (s Set[T]) Difference(other Set[T]) Set[T] {
    result := make(Set[T])
    for k := range s {
        if !other.Contains(k) {
            result[k] = struct{}{}
        }
    }
    return result
}

// Equals checks if two sets contain the same elements
func (s Set[T]) Equals(other Set[T]) bool {
    if len(s) != len(other) {
        return false
    }
    for k := range s {
        if !other.Contains(k) {
            return false
        }
    }
    return true
}
