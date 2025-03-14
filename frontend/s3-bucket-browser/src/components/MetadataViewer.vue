<script setup>
import { computed, ref } from 'vue'

const props = defineProps({
  metadata: {
    type: Object,
    default: null
  }
})

// Toggle for showing raw JSON
const showRawJson = ref(false)

// Toggle raw JSON visibility
const toggleRawJson = () => {
  showRawJson.value = !showRawJson.value
}

// Format date
const formatDate = (date) => {
  if (!date) return ''
  
  // Check if it's a Unix timestamp (number)
  if (typeof date === 'number') {
    // Convert seconds to milliseconds if needed
    const milliseconds = date > 10000000000 ? date : date * 1000
    return new Date(milliseconds).toLocaleString()
  }
  
  // Handle string date
  try {
    const d = new Date(date)
    return d.toLocaleString()
  } catch (e) {
    return date.toString()
  }
}

// Format file size
const formatFileSize = (bytes) => {
  if (!bytes || isNaN(bytes) || bytes === 0) return 'Unknown'
  
  const k = 1024
  const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

// Get status color
const getStatusColor = (status) => {
  if (!status) return 'gray'
  
  switch (status.toLowerCase()) {
    case 'success':
    case 'completed':
      return 'green'
    case 'error':
    case 'failed':
      return 'red'
    case 'pending':
    case 'processing':
      return 'yellow'
    default:
      return 'gray'
  }
}

// Check if metadata is available
const hasMetadata = computed(() => {
  return props.metadata !== null
})
</script>

<template>
  <div>
    <div v-if="!hasMetadata" class="text-center p-4 text-gray-500">
      <p>Select a file to view its metadata</p>
    </div>
    
    <div v-else class="space-y-4">
      <!-- Basic info -->
      <div class="bg-gray-50 p-4 rounded-lg">
        <h3 class="text-lg font-medium text-gray-900 mb-2">Basic Information</h3>
        
        <div class="grid grid-cols-1 gap-2">
          <div class="flex justify-between">
            <span class="text-sm font-medium text-gray-500">File Name:</span>
            <span class="text-sm text-gray-900">{{ metadata.file_name || metadata.fileName || 'Unknown' }}</span>
          </div>
          
          <div class="flex justify-between">
            <span class="text-sm font-medium text-gray-500">File Size:</span>
            <span class="text-sm text-gray-900">{{ formatFileSize(metadata.file_size || metadata.fileSize) }}</span>
          </div>
          
          <div class="flex justify-between">
            <span class="text-sm font-medium text-gray-500">Status:</span>
            <span v-if="metadata.status" 
              class="px-2 inline-flex text-xs leading-5 font-semibold rounded-full"
              :class="`bg-${getStatusColor(metadata.status)}-100 text-${getStatusColor(metadata.status)}-800`"
            >
              {{ metadata.status }}
            </span>
            <span v-else class="text-sm text-gray-500">Unknown</span>
          </div>
        </div>
      </div>
      
      <!-- Solana info -->
      <div class="bg-gray-50 p-4 rounded-lg">
        <h3 class="text-lg font-medium text-gray-900 mb-2">Solana Information</h3>
        
        <div class="grid grid-cols-1 gap-2">
          <div class="flex justify-between">
            <span class="text-sm font-medium text-gray-500">Solana Version:</span>
            <span class="text-sm text-gray-900">{{ metadata.solana_version || 'Unknown' }}</span>
          </div>
          
          <div class="flex justify-between">
            <span class="text-sm font-medium text-gray-500">Feature Set:</span>
            <span class="text-sm text-gray-900">{{ metadata.solana_feature_set || 'Unknown' }}</span>
          </div>
          
          <div class="flex justify-between">
            <span class="text-sm font-medium text-gray-500">Slot:</span>
            <span class="text-sm text-gray-900">{{ metadata.slot || 'Unknown' }}</span>
          </div>
          
          <div v-if="metadata.slot_range" class="flex justify-between">
            <span class="text-sm font-medium text-gray-500">Slot Range:</span>
            <span class="text-sm text-gray-900">{{ metadata.slot_range }}</span>
          </div>
          
          <div v-if="metadata.node" class="flex justify-between">
            <span class="text-sm font-medium text-gray-500">Node:</span>
            <span class="text-sm text-gray-900 truncate max-w-[200px]" :title="metadata.node">
              {{ metadata.node }}
            </span>
          </div>
          
          <div class="flex justify-between">
            <span class="text-sm font-medium text-gray-500">Hash:</span>
            <span class="text-sm text-gray-900 truncate max-w-[200px]" :title="metadata.hash">
              {{ metadata.hash || 'Unknown' }}
            </span>
          </div>
        </div>
      </div>
      
      <!-- Timestamps -->
      <div class="bg-gray-50 p-4 rounded-lg">
        <h3 class="text-lg font-medium text-gray-900 mb-2">Timestamps</h3>
        
        <div class="grid grid-cols-1 gap-2">
          <div class="flex justify-between">
            <span class="text-sm font-medium text-gray-500">Timestamp:</span>
            <span class="text-sm text-gray-900">
              {{ metadata.timestamp_human || formatDate(metadata.timestamp) || 'Unknown' }}
            </span>
          </div>
          
          <div class="flex justify-between">
            <span class="text-sm font-medium text-gray-500">Uploaded At:</span>
            <span class="text-sm text-gray-900">{{ formatDate(metadata.uploaded_at) || 'Unknown' }}</span>
          </div>
          
          <div class="flex justify-between">
            <span class="text-sm font-medium text-gray-500">Uploaded By:</span>
            <span class="text-sm text-gray-900">{{ metadata.uploaded_by || 'Unknown' }}</span>
          </div>
        </div>
      </div>
      
      <!-- Raw JSON -->
      <div class="bg-gray-50 p-4 rounded-lg">
        <div class="flex justify-between items-center mb-2">
          <h3 class="text-lg font-medium text-gray-900">Raw JSON</h3>
          <button 
            @click="toggleRawJson" 
            class="text-xs text-blue-600 hover:text-blue-800 underline"
            :title="showRawJson ? 'Hide raw JSON' : 'Show raw JSON'"
          >
            {{ showRawJson ? 'Hide' : 'Show' }}
          </button>
        </div>
        
        <pre v-if="showRawJson" class="bg-gray-800 text-white p-4 rounded-lg text-xs overflow-auto max-h-60">{{ JSON.stringify(metadata, null, 2) }}</pre>
      </div>
    </div>
  </div>
</template> 