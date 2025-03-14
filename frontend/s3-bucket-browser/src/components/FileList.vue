<script setup>
import { ref, computed, watch } from 'vue'

const props = defineProps({
  files: {
    type: Array,
    required: true
  }
})

const emit = defineEmits(['file-select'])

const sortBy = ref('LastModified')
const sortDirection = ref('desc')
const currentPage = ref(1)
const itemsPerPage = ref(15)
const selectedFile = ref(null)

// Format file size
const formatFileSize = (bytes) => {
  if (bytes === 0) return '0 Bytes'
  
  const k = 1024
  const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

// Format date
const formatDate = (date) => {
  if (!date) return ''
  
  const d = new Date(date)
  return d.toLocaleString()
}

// Get file name from path
const getFileName = (path) => {
  return path.split('/').pop()
}

// Get file type class
const getFileTypeClass = (file) => {
  if (file.IsTarGz) {
    return 'bg-yellow-100 text-yellow-800'
  } else if (file.IsMetadata) {
    return 'bg-blue-100 text-blue-800'
  } else {
    return 'bg-gray-100 text-gray-800'
  }
}

// Sort files
const sortedFiles = computed(() => {
  return [...props.files].sort((a, b) => {
    let aValue = a[sortBy.value]
    let bValue = b[sortBy.value]
    
    // Handle dates
    if (sortBy.value === 'LastModified') {
      aValue = new Date(aValue).getTime()
      bValue = new Date(bValue).getTime()
    }
    
    // Handle strings
    if (typeof aValue === 'string') {
      aValue = aValue.toLowerCase()
      bValue = bValue.toLowerCase()
    }
    
    if (sortDirection.value === 'asc') {
      return aValue > bValue ? 1 : -1
    } else {
      return aValue < bValue ? 1 : -1
    }
  })
})

// Paginate files
const paginatedFiles = computed(() => {
  const start = (currentPage.value - 1) * itemsPerPage.value
  const end = start + itemsPerPage.value
  
  return sortedFiles.value.slice(start, end)
})

// Total pages
const totalPages = computed(() => {
  return Math.ceil(props.files.length / itemsPerPage.value) || 1
})

// Page numbers to display
const displayedPages = computed(() => {
  const totalPagesCount = totalPages.value
  const current = currentPage.value
  const delta = 2 // Number of pages to show before and after current page
  
  if (totalPagesCount <= 5) {
    return Array.from({ length: totalPagesCount }, (_, i) => i + 1)
  }
  
  let pages = [1]
  
  // Calculate range
  const leftBound = Math.max(2, current - delta)
  const rightBound = Math.min(totalPagesCount - 1, current + delta)
  
  // Add ellipsis if needed
  if (leftBound > 2) {
    pages.push('...')
  }
  
  // Add pages in range
  for (let i = leftBound; i <= rightBound; i++) {
    pages.push(i)
  }
  
  // Add ellipsis if needed
  if (rightBound < totalPagesCount - 1) {
    pages.push('...')
  }
  
  // Add last page
  if (totalPagesCount > 1) {
    pages.push(totalPagesCount)
  }
  
  return pages
})

// Handle sort
const handleSort = (column) => {
  if (sortBy.value === column) {
    sortDirection.value = sortDirection.value === 'asc' ? 'desc' : 'asc'
  } else {
    sortBy.value = column
    sortDirection.value = 'asc'
  }
}

// Handle file click
const handleFileClick = (file) => {
  selectedFile.value = file.Key
  emit('file-select', file)
}

// Handle page change
const changePage = (page) => {
  if (typeof page === 'number') {
    currentPage.value = page
  }
}

// Reset pagination when files change
watch(() => props.files.length, () => {
  currentPage.value = 1
})
</script>

<template>
  <div>
    <div class="overflow-x-auto">
      <table class="min-w-full divide-y divide-gray-200 table-fixed">
        <thead class="bg-gray-50">
          <tr>
            <th 
              @click="handleSort('Key')" 
              class="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer w-2/5"
            >
              <div class="flex items-center">
                <span>Name</span>
                <span v-if="sortBy === 'Key'" class="ml-1">
                  <svg v-if="sortDirection === 'asc'" class="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
                    <path fill-rule="evenodd" d="M14.707 12.707a1 1 0 01-1.414 0L10 9.414l-3.293 3.293a1 1 0 01-1.414-1.414l4-4a1 1 0 011.414 0l4 4a1 1 0 010 1.414z" clip-rule="evenodd" />
                  </svg>
                  <svg v-else class="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
                    <path fill-rule="evenodd" d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z" clip-rule="evenodd" />
                  </svg>
                </span>
              </div>
            </th>
            <th 
              @click="handleSort('Size')" 
              class="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer w-1/5"
            >
              <div class="flex items-center">
                <span>Size</span>
                <span v-if="sortBy === 'Size'" class="ml-1">
                  <svg v-if="sortDirection === 'asc'" class="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
                    <path fill-rule="evenodd" d="M14.707 12.707a1 1 0 01-1.414 0L10 9.414l-3.293 3.293a1 1 0 01-1.414-1.414l4-4a1 1 0 011.414 0l4 4a1 1 0 010 1.414z" clip-rule="evenodd" />
                  </svg>
                  <svg v-else class="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
                    <path fill-rule="evenodd" d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z" clip-rule="evenodd" />
                  </svg>
                </span>
              </div>
            </th>
            <th 
              @click="handleSort('LastModified')" 
              class="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer w-1/5"
            >
              <div class="flex items-center">
                <span>Last Modified</span>
                <span v-if="sortBy === 'LastModified'" class="ml-1">
                  <svg v-if="sortDirection === 'asc'" class="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
                    <path fill-rule="evenodd" d="M14.707 12.707a1 1 0 01-1.414 0L10 9.414l-3.293 3.293a1 1 0 01-1.414-1.414l4-4a1 1 0 011.414 0l4 4a1 1 0 010 1.414z" clip-rule="evenodd" />
                  </svg>
                  <svg v-else class="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
                    <path fill-rule="evenodd" d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z" clip-rule="evenodd" />
                  </svg>
                </span>
              </div>
            </th>
            <th 
              class="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider w-1/5"
            >
              Type
            </th>
          </tr>
        </thead>
        <tbody class="bg-white divide-y divide-gray-200">
          <tr 
            v-for="file in paginatedFiles" 
            :key="file.Key" 
            @click="handleFileClick(file)"
            class="hover:bg-gray-50 cursor-pointer transition-colors duration-150"
            :class="{ 'bg-blue-50': selectedFile === file.Key }"
          >
            <td class="px-3 py-2 whitespace-nowrap text-sm font-medium text-gray-900 truncate">
              <div class="flex items-center">
                <span v-if="file.IsTarGz" class="mr-2 text-yellow-500">
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4"></path>
                  </svg>
                </span>
                <span v-else-if="file.IsMetadata" class="mr-2 text-blue-500">
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"></path>
                  </svg>
                </span>
                <span v-else class="mr-2 text-gray-500">
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z"></path>
                  </svg>
                </span>
                <span class="truncate" :title="file.Key">{{ getFileName(file.Key) }}</span>
              </div>
            </td>
            <td class="px-3 py-2 whitespace-nowrap text-sm text-gray-500">
              {{ formatFileSize(file.Size) }}
            </td>
            <td class="px-3 py-2 whitespace-nowrap text-sm text-gray-500">
              {{ formatDate(file.LastModified) }}
            </td>
            <td class="px-3 py-2 whitespace-nowrap text-sm text-gray-500">
              <span 
                class="px-2 inline-flex text-xs leading-5 font-semibold rounded-full"
                :class="getFileTypeClass(file)"
              >
                {{ file.IsTarGz ? 'Archive' : file.IsMetadata ? 'Metadata' : 'File' }}
              </span>
            </td>
          </tr>
          <tr v-if="paginatedFiles.length === 0">
            <td colspan="4" class="px-4 py-4 text-center text-sm text-gray-500">
              No files found
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    
    <!-- Pagination -->
    <div class="flex items-center justify-between border-t border-gray-200 bg-white px-4 py-3 sm:px-6 mt-4">
      <div class="flex flex-1 justify-between sm:hidden">
        <button
          @click="changePage(currentPage - 1)"
          :disabled="currentPage === 1"
          class="relative inline-flex items-center rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
          :class="{ 'opacity-50 cursor-not-allowed': currentPage === 1 }"
        >
          Previous
        </button>
        <button
          @click="changePage(currentPage + 1)"
          :disabled="currentPage === totalPages"
          class="relative ml-3 inline-flex items-center rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
          :class="{ 'opacity-50 cursor-not-allowed': currentPage === totalPages }"
        >
          Next
        </button>
      </div>
      <div class="hidden sm:flex sm:flex-1 sm:items-center sm:justify-between">
        <div>
          <p class="text-sm text-gray-700">
            Showing
            <span class="font-medium">{{ props.files.length ? (currentPage - 1) * itemsPerPage + 1 : 0 }}</span>
            to
            <span class="font-medium">{{ Math.min(currentPage * itemsPerPage, props.files.length) }}</span>
            of
            <span class="font-medium">{{ props.files.length }}</span>
            results
          </p>
        </div>
        <div>
          <nav class="isolate inline-flex -space-x-px rounded-md shadow-sm" aria-label="Pagination">
            <button
              @click="changePage(currentPage - 1)"
              :disabled="currentPage === 1"
              class="relative inline-flex items-center rounded-l-md px-2 py-2 text-gray-400 ring-1 ring-inset ring-gray-300 hover:bg-gray-50 focus:z-20 focus:outline-offset-0"
              :class="{ 'opacity-50 cursor-not-allowed': currentPage === 1 }"
            >
              <span class="sr-only">Previous</span>
              <svg class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                <path fill-rule="evenodd" d="M12.79 5.23a.75.75 0 01-.02 1.06L8.832 10l3.938 3.71a.75.75 0 11-1.04 1.08l-4.5-4.25a.75.75 0 010-1.08l4.5-4.25a.75.75 0 011.06.02z" clip-rule="evenodd" />
              </svg>
            </button>
            <template v-for="(page, index) in displayedPages" :key="index">
              <span
                v-if="page === '...'"
                class="relative inline-flex items-center px-4 py-2 text-sm font-semibold text-gray-700 ring-1 ring-inset ring-gray-300"
              >
                ...
              </span>
              <button
                v-else
                @click="changePage(page)"
                class="relative inline-flex items-center px-4 py-2 text-sm font-semibold ring-1 ring-inset ring-gray-300 focus:z-20 focus:outline-offset-0"
                :class="currentPage === page ? 'bg-blue-600 text-white hover:bg-blue-700' : 'text-gray-900 hover:bg-gray-50'"
              >
                {{ page }}
              </button>
            </template>
            <button
              @click="changePage(currentPage + 1)"
              :disabled="currentPage === totalPages"
              class="relative inline-flex items-center rounded-r-md px-2 py-2 text-gray-400 ring-1 ring-inset ring-gray-300 hover:bg-gray-50 focus:z-20 focus:outline-offset-0"
              :class="{ 'opacity-50 cursor-not-allowed': currentPage === totalPages }"
            >
              <span class="sr-only">Next</span>
              <svg class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                <path fill-rule="evenodd" d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z" clip-rule="evenodd" />
              </svg>
            </button>
          </nav>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.truncate {
  max-width: 250px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style> 