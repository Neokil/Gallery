// Upload functionality
const uploadArea = document.getElementById('upload-area');
const fileInput = document.getElementById('file-input');
const uploadForm = document.getElementById('upload-form');
const uploadContent = document.getElementById('upload-content');

// Click to select files
uploadArea.addEventListener('click', () => {
    if (!uploadArea.classList.contains('uploading')) {
        fileInput.click();
    }
});

// Drag and drop
uploadArea.addEventListener('dragover', (e) => {
    e.preventDefault();
    if (!uploadArea.classList.contains('uploading')) {
        uploadArea.classList.add('dragover');
    }
});

uploadArea.addEventListener('dragleave', () => {
    uploadArea.classList.remove('dragover');
});

uploadArea.addEventListener('drop', (e) => {
    e.preventDefault();
    uploadArea.classList.remove('dragover');
    
    if (!uploadArea.classList.contains('uploading')) {
        const files = e.dataTransfer.files;
        fileInput.files = files;
        uploadFiles();
    }
});

// File input change - auto upload
fileInput.addEventListener('change', () => {
    if (fileInput.files.length > 0) {
        uploadFiles();
    }
});

function uploadFiles() {
    const files = fileInput.files;
    if (files.length === 0) return;
    
    // Show uploading state
    uploadArea.classList.add('uploading');
    const count = files.length;
    uploadContent.innerHTML = `
        <div class="upload-spinner"></div>
        <p>Uploading ${count} photo${count > 1 ? 's' : ''}...</p>
    `;
    
    // Create FormData and upload
    const formData = new FormData();
    for (let i = 0; i < files.length; i++) {
        formData.append('photos', files[i]);
    }
    
    fetch('/upload', {
        method: 'POST',
        body: formData
    })
    .then(response => {
        if (response.ok) {
            // Success - reload the page to show new photos
            window.location.reload();
        } else {
            throw new Error('Upload failed');
        }
    })
    .catch(error => {
        console.error('Upload error:', error);
        // Reset upload area on error
        resetUploadArea();
        alert('Upload failed. Please try again.');
    });
}

function resetUploadArea() {
    uploadArea.classList.remove('uploading');
    uploadContent.innerHTML = `
        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
            <polyline points="7,10 12,15 17,10"></polyline>
            <line x1="12" y1="15" x2="12" y2="3"></line>
        </svg>
        <p>Drag photos here or click to select</p>
    `;
    fileInput.value = '';
}

// Modal functionality
function openModal(imageSrc) {
    const modal = document.getElementById('modal');
    const modalImg = document.getElementById('modal-img');
    
    modal.style.display = 'block';
    modalImg.src = imageSrc;
}

function closeModal() {
    document.getElementById('modal').style.display = 'none';
}

// Close modal with Escape key
document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
        closeModal();
    }
});

// Prevent form submission (we handle uploads via fetch now)
uploadForm.addEventListener('submit', (e) => {
    e.preventDefault();
});