<script setup>
import { ref, reactive, onMounted, onUnmounted, watch } from 'vue'

const emit = defineEmits(['filter-change'])

const filters = reactive({
  solanaVersion: '',
  solanaFeatureSet: '',
  status: '',
  uploadedBy: '',
  node: '',
  slotRange: '',
  startTime: '',
  endTime: '',
  minSlot: '',
  maxSlot: '',
  searchTerm: '',
  page: 1,
  pageSize: 20
})

// Available options (will be populated from API)
const statusOptions = ref([{ value: '', label: 'All Statuses' }])
const solanaVersionOptions = ref([{ value: '', label: 'All Versions' }])
const uploadedByOptions = ref([{ value: '', label: 'All Users' }])
const nodeOptions = ref([{ value: '', label: 'All Nodes' }])
const slotRangeOptions = ref([{ value: '', label: 'All Slot Ranges' }])

const isFiltersApplied = ref(false)
const loading = ref(false)
const error = ref('')

// Apply filters
const applyFilters = () => {
  try {
    console.log('Applying filters:', filters)
    
    // Create a copy of the filters to send to the parent
    const filtersToSend = { ...filters, page: 1 }
    
    // Make sure we're using the correct parameter names that the backend expects
    if (filters.solanaVersion) filtersToSend.solanaVersion = filters.solanaVersion
    if (filters.status) filtersToSend.status = filters.status
    if (filters.uploadedBy) filtersToSend.uploadedBy = filters.uploadedBy
    if (filters.node) filtersToSend.node = filters.node
    if (filters.slotRange) filtersToSend.slotRange = filters.slotRange
    if (filters.minSlot) filtersToSend.minSlot = filters.minSlot
    if (filters.maxSlot) filtersToSend.maxSlot = filters.maxSlot
    if (filters.startTime) filtersToSend.startTime = filters.startTime
    if (filters.endTime) filtersToSend.endTime = filters.endTime
    if (filters.searchTerm) filtersToSend.searchTerm = filters.searchTerm
    
    console.log('Sending filters to parent:', filtersToSend)
    
    isFiltersApplied.value = true
    emit('filter-change', filtersToSend) // Reset to page 1 when applying filters
  } catch (err) {
    console.error('Error applying filters:', err)
    error.value = 'Failed to apply filters. Please try again.'
  }
}

// Reset filters
const resetFilters = () => {
  try {
    console.log('Resetting filters')
    Object.keys(filters).forEach(key => {
      if (key !== 'page' && key !== 'pageSize') {
        filters[key] = ''
      }
    })
    
    filters.page = 1
    isFiltersApplied.value = false
    emit('filter-change', { ...filters })
  } catch (err) {
    console.error('Error resetting filters:', err)
    error.value = 'Failed to reset filters. Please try again.'
  }
}

// Handle input changes for dynamic filtering
const handleInputChange = () => {
  if (isFiltersApplied.value) {
    applyFilters()
  }
}

// Fetch filter options from API
const fetchFilterOptions = async () => {
  loading.value = true
  error.value = ''
  
  try {
    console.log('Fetching filter options from API...')
    const apiUrl = '/api/metadata/options'
    console.log(`API URL: ${apiUrl}`)
    
    const response = await fetch(apiUrl)
    
    if (!response.ok) {
      const errorText = await response.text()
      console.error('API response not OK:', response.status, errorText)
      throw new Error(`Failed to fetch filter options: ${response.status} ${response.statusText}`)
    }
    
    const data = await response.json()
    console.log('Filter options from API:', data)
    
    // Check if we got valid data
    if (!data || typeof data !== 'object') {
      console.error('Invalid data received:', data)
      throw new Error('Invalid data received from API')
    }
    
    // Update status options
    if (data.statuses && Array.isArray(data.statuses)) {
      console.log(`Received ${data.statuses.length} status options`)
      statusOptions.value = [
        { value: '', label: 'All Statuses' },
        ...data.statuses.map(status => ({ value: status, label: status }))
      ]
    } else {
      console.warn('No status options received or invalid format')
    }
    
    // Update Solana version options
    if (data.solanaVersions && Array.isArray(data.solanaVersions)) {
      console.log(`Received ${data.solanaVersions.length} Solana version options`)
      solanaVersionOptions.value = [
        { value: '', label: 'All Versions' },
        ...data.solanaVersions.map(version => ({ value: version, label: version }))
      ]
    } else {
      console.warn('No Solana version options received or invalid format')
    }
    
    // Update uploaded by options
    if (data.uploadedBy && Array.isArray(data.uploadedBy)) {
      console.log(`Received ${data.uploadedBy.length} uploader options`)
      uploadedByOptions.value = [
        { value: '', label: 'All Users' },
        ...data.uploadedBy.map(user => ({ value: user, label: user }))
      ]
    } else {
      console.warn('No uploader options received or invalid format')
    }
    
    // Update node options
    if (data.nodes && Array.isArray(data.nodes)) {
      console.log(`Received ${data.nodes.length} node options`)
      nodeOptions.value = [
        { value: '', label: 'All Nodes' },
        ...data.nodes.map(node => ({ value: node, label: node }))
      ]
    } else {
      console.warn('No node options received or invalid format')
    }
    
    // Update slot range options
    if (data.slotRanges && Array.isArray(data.slotRanges)) {
      console.log(`Received ${data.slotRanges.length} slot range options`)
      slotRangeOptions.value = [
        { value: '', label: 'All Slot Ranges' },
        ...data.slotRanges.map(range => ({ value: range, label: range }))
      ]
    } else {
      console.warn('No slot range options received or invalid format')
    }
    
    // If we have no options, try to trigger a reindex
    if ((!data.solanaVersions || data.solanaVersions.length === 0) && 
        (!data.statuses || data.statuses.length === 0)) {
      console.log('No options available, triggering manual reindex...')
      triggerReindex()
    }
  } catch (err) {
    console.error('Failed to fetch filter options:', err)
    error.value = err.message
  } finally {
    loading.value = false
  }
}

// Trigger a manual reindex
const triggerReindex = async () => {
  try {
    console.log('Triggering manual reindex...')
    const apiUrl = '/api/debug/reindex'
    console.log(`Reindex API URL: ${apiUrl}`)
    
    const response = await fetch(apiUrl)
    
    if (!response.ok) {
      const errorText = await response.text()
      console.error('Reindex API response not OK:', response.status, errorText)
      throw new Error(`Failed to trigger reindex: ${response.status} ${response.statusText}`)
    }
    
    const data = await response.json()
    console.log('Reindex response:', data)
    
    // Wait a bit and then try to fetch options again
    setTimeout(() => {
      console.log('Retrying fetch after reindex...')
      fetchFilterOptions()
    }, 5000) // Wait 5 seconds
  } catch (err) {
    console.error('Failed to trigger reindex:', err)
    error.value = `Failed to trigger reindex: ${err.message}`
  }
}

// Add a button to manually trigger reindex
const manualReindex = () => {
  triggerReindex()
}

// Refresh filter options periodically
let refreshTimer = null

const startRefreshTimer = () => {
  // Refresh filter options every 5 minutes
  refreshTimer = setInterval(() => {
    fetchFilterOptions()
  }, 5 * 60 * 1000)
}

// Initialize component
onMounted(() => {
  fetchFilterOptions()
  startRefreshTimer()
})

// Clean up on component unmount
onUnmounted(() => {
  if (refreshTimer) {
    clearInterval(refreshTimer)
  }
})
</script>

<template>
  <div class="bg-white shadow rounded-lg p-4">
    <div class="flex justify-between items-center mb-4">
      <h2 class="text-xl font-semibold">Search & Filters</h2>
      <button 
        @click="manualReindex" 
        class="text-xs text-blue-600 hover:text-blue-800 underline"
        title="Refresh filter options from server"
      >
        Refresh Options
      </button>
    </div>
    
    <div v-if="loading" class="flex justify-center my-4">
      <div class="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-500"></div>
    </div>
    
    <div v-if="error" class="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-4">
      {{ error }}
      <button 
        @click="fetchFilterOptions" 
        class="ml-2 text-red-700 hover:text-red-900 underline"
      >
        Retry
      </button>
    </div>
    
    <div class="space-y-4">
      <!-- Search -->
      <div>
        <label for="search" class="block text-sm font-medium text-gray-700">Search</label>
        <div class="mt-1 relative rounded-md shadow-sm">
          <input
            type="text"
            id="search"
            v-model="filters.searchTerm"
            @input="handleInputChange"
            class="focus:ring-blue-500 focus:border-blue-500 block w-full pl-3 pr-10 py-2 sm:text-sm border-gray-300 rounded-md"
            placeholder="Search by version, hash, status..."
          />
          <div class="absolute inset-y-0 right-0 pr-3 flex items-center pointer-events-none">
            <svg class="h-4 w-4 text-gray-400" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
              <path fill-rule="evenodd" d="M8 4a4 4 0 100 8 4 4 0 000-8zM2 8a6 6 0 1110.89 3.476l4.817 4.817a1 1 0 01-1.414 1.414l-4.816-4.816A6 6 0 012 8z" clip-rule="evenodd" />
            </svg>
          </div>
        </div>
      </div>
      
      <!-- Solana Version -->
      <div>
        <label for="solana-version" class="block text-sm font-medium text-gray-700">
          Solana Version
          <span class="text-xs text-gray-500 ml-1">({{ solanaVersionOptions.length - 1 }} available)</span>
        </label>
        <select
          id="solana-version"
          v-model="filters.solanaVersion"
          @change="handleInputChange"
          class="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm rounded-md"
        >
          <option v-for="option in solanaVersionOptions" :key="option.value" :value="option.value">
            {{ option.label }}
          </option>
        </select>
      </div>
      
      <!-- Status -->
      <div>
        <label for="status" class="block text-sm font-medium text-gray-700">
          Status
          <span class="text-xs text-gray-500 ml-1">({{ statusOptions.length - 1 }} available)</span>
        </label>
        <select
          id="status"
          v-model="filters.status"
          @change="handleInputChange"
          class="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm rounded-md"
        >
          <option v-for="option in statusOptions" :key="option.value" :value="option.value">
            {{ option.label }}
          </option>
        </select>
      </div>
      
      <!-- Uploaded By -->
      <div>
        <label for="uploaded-by" class="block text-sm font-medium text-gray-700">
          Uploaded By
          <span class="text-xs text-gray-500 ml-1">({{ uploadedByOptions.length - 1 }} available)</span>
        </label>
        <select
          id="uploaded-by"
          v-model="filters.uploadedBy"
          @change="handleInputChange"
          class="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm rounded-md"
        >
          <option v-for="option in uploadedByOptions" :key="option.value" :value="option.value">
            {{ option.label }}
          </option>
        </select>
      </div>
      
      <!-- Node -->
      <div>
        <label for="node" class="block text-sm font-medium text-gray-700">
          Node
          <span class="text-xs text-gray-500 ml-1">({{ nodeOptions.length - 1 }} available)</span>
        </label>
        <select
          id="node"
          v-model="filters.node"
          @change="handleInputChange"
          class="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm rounded-md"
        >
          <option v-for="option in nodeOptions" :key="option.value" :value="option.value">
            {{ option.label }}
          </option>
        </select>
      </div>
      
      <!-- Slot Range -->
      <div>
        <label for="slot-range" class="block text-sm font-medium text-gray-700">
          Slot Range
          <span class="text-xs text-gray-500 ml-1">({{ slotRangeOptions.length - 1 }} available)</span>
        </label>
        <select
          id="slot-range"
          v-model="filters.slotRange"
          @change="handleInputChange"
          class="mt-1 block w-full pl-3 pr-10 py-2 text-base border-gray-300 focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm rounded-md"
        >
          <option v-for="option in slotRangeOptions" :key="option.value" :value="option.value">
            {{ option.label }}
          </option>
        </select>
      </div>
      
      <!-- Slot Range (Custom) -->
      <div class="grid grid-cols-2 gap-4">
        <div>
          <label for="min-slot" class="block text-sm font-medium text-gray-700">Min Slot</label>
          <input
            type="number"
            id="min-slot"
            v-model="filters.minSlot"
            @input="handleInputChange"
            class="mt-1 focus:ring-blue-500 focus:border-blue-500 block w-full shadow-sm sm:text-sm border-gray-300 rounded-md"
            placeholder="Min"
          />
        </div>
        <div>
          <label for="max-slot" class="block text-sm font-medium text-gray-700">Max Slot</label>
          <input
            type="number"
            id="max-slot"
            v-model="filters.maxSlot"
            @input="handleInputChange"
            class="mt-1 focus:ring-blue-500 focus:border-blue-500 block w-full shadow-sm sm:text-sm border-gray-300 rounded-md"
            placeholder="Max"
          />
        </div>
      </div>
      
      <!-- Date Range -->
      <div class="grid grid-cols-2 gap-4">
        <div>
          <label for="start-time" class="block text-sm font-medium text-gray-700">Start Date</label>
          <input
            type="date"
            id="start-time"
            v-model="filters.startTime"
            @input="handleInputChange"
            class="mt-1 focus:ring-blue-500 focus:border-blue-500 block w-full shadow-sm sm:text-sm border-gray-300 rounded-md"
          />
        </div>
        <div>
          <label for="end-time" class="block text-sm font-medium text-gray-700">End Date</label>
          <input
            type="date"
            id="end-time"
            v-model="filters.endTime"
            @input="handleInputChange"
            class="mt-1 focus:ring-blue-500 focus:border-blue-500 block w-full shadow-sm sm:text-sm border-gray-300 rounded-md"
          />
        </div>
      </div>
      
      <!-- Buttons -->
      <div class="flex space-x-4 pt-4">
        <button
          @click="applyFilters"
          class="inline-flex justify-center py-2 px-4 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
        >
          Apply Filters
        </button>
        <button
          @click="resetFilters"
          class="inline-flex justify-center py-2 px-4 border border-gray-300 shadow-sm text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
          :disabled="!isFiltersApplied"
          :class="{ 'opacity-50 cursor-not-allowed': !isFiltersApplied }"
        >
          Reset
        </button>
      </div>
    </div>
  </div>
</template> 