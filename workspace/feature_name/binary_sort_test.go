package feature_name

import (
    "testing"
)

// BinarySearchTest tests the binary search function
func BinarySearchTest(t *testing.T) {
    arr := []int{1, 3, 5, 7, 9, 11, 13, 15, 17, 19}
    target := 7
    expected := 3
    result := BinarySearch(arr, target)
    if result != expected {
        t.Errorf("Expected %d, but got %d", expected, result)
    }
    
    // Test case for target not found in the array
    target = 10
    expected = -1
    result = BinarySearch(arr, target)
    if result != expected {
        t.Errorf("Expected %d, but got %d", expected, result)
    }
}