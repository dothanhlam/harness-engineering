package feature_name

// BinarySearch represents a function to perform binary search on a sorted array
func BinarySearch(arr []int, target int) int {
    low, high := 0, len(arr)-1
    for low <= high {
        mid := low + (high-low)/2
        if arr[mid] == target {
            return mid
        }
        if arr[mid] < target {
            low = mid + 1
        }
        if arr[mid] > target {
            high = mid - 1
        }
    }
    return -1
}