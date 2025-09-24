#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/socket.h>
#include <linux/if.h>
#include <linux/ethtool.h>
#include <linux/sockios.h>
#include <sys/ioctl.h>
#include <unistd.h>
#include <openssl/evp.h>
#include <openssl/aes.h>

// Check if interface supports specific hardware offload feature
int check_hardware_offload_support(const char *ifname, int feature) {
    struct ifreq ifr;
    struct ethtool_value edata;
    int sock;
    int ret = -1;

    sock = socket(AF_INET, SOCK_DGRAM, 0);
    if (sock < 0) {
        return -1;
    }

    memset(&ifr, 0, sizeof(ifr));
    strncpy(ifr.ifr_name, ifname, IFNAMSIZ - 1);

    edata.cmd = ETHTOOL_GTXCSUM; // Default to TX checksum
    switch (feature) {
        case 0: edata.cmd = ETHTOOL_GTXCSUM; break;   // TX checksum
        case 1: edata.cmd = ETHTOOL_GRXCSUM; break;   // RX checksum
        case 2: edata.cmd = ETHTOOL_GTSO; break;      // TSO
        case 3: edata.cmd = ETHTOOL_GGSO; break;      // GSO
        case 4: edata.cmd = ETHTOOL_GGRO; break;      // GRO
        case 5: edata.cmd = ETHTOOL_GSG; break;       // Scatter-gather
        default: break;
    }

    ifr.ifr_data = (caddr_t)&edata;
    if (ioctl(sock, SIOCETHTOOL, &ifr) == 0) {
        ret = edata.data;
    }

    close(sock);
    return ret;
}

// Enable or disable hardware offload feature
int enable_hardware_offload(const char *ifname, int feature, int enable) {
    struct ifreq ifr;
    struct ethtool_value edata;
    int sock;
    int ret = -1;

    sock = socket(AF_INET, SOCK_DGRAM, 0);
    if (sock < 0) {
        return -1;
    }

    memset(&ifr, 0, sizeof(ifr));
    strncpy(ifr.ifr_name, ifname, IFNAMSIZ - 1);

    edata.cmd = ETHTOOL_STXCSUM; // Default to set TX checksum
    edata.data = enable ? 1 : 0;

    switch (feature) {
        case 0: edata.cmd = ETHTOOL_STXCSUM; break;   // Set TX checksum
        case 1: edata.cmd = ETHTOOL_SRXCSUM; break;   // Set RX checksum
        case 2: edata.cmd = ETHTOOL_STSO; break;      // Set TSO
        case 3: edata.cmd = ETHTOOL_SGSO; break;      // Set GSO
        case 4: edata.cmd = ETHTOOL_SGRO; break;      // Set GRO
        case 5: edata.cmd = ETHTOOL_SSG; break;       // Set Scatter-gather
        default: break;
    }

    ifr.ifr_data = (caddr_t)&edata;
    if (ioctl(sock, SIOCETHTOOL, &ifr) == 0) {
        ret = 0;
        printf("Offload: %s feature %d on interface %s\n", 
               enable ? "Enabled" : "Disabled", feature, ifname);
    } else {
        printf("Offload: Failed to %s feature %d on interface %s\n", 
               enable ? "enable" : "disable", feature, ifname);
    }

    close(sock);
    return ret;
}

// Perform hardware checksum calculation
int hardware_checksum_offload(void *data, int len, int type) {
    // This would use hardware-specific APIs or kernel interfaces
    // For demonstration, we'll use a simple software implementation
    
    if (type == 0) { // CRC32
        unsigned int crc = 0xFFFFFFFF;
        unsigned char *buf = (unsigned char *)data;
        
        for (int i = 0; i < len; i++) {
            crc ^= buf[i];
            for (int j = 0; j < 8; j++) {
                if (crc & 1) {
                    crc = (crc >> 1) ^ 0xEDB88320;
                } else {
                    crc >>= 1;
                }
            }
        }
        
        return crc ^ 0xFFFFFFFF;
    }
    
    return -1;
}

// Perform hardware crypto encryption
int hardware_crypto_encrypt(void *plaintext, int len, void *key, int keylen, void *ciphertext) {
    EVP_CIPHER_CTX *ctx;
    int ciphertext_len;
    int final_len;

    // Create and initialize the context
    ctx = EVP_CIPHER_CTX_new();
    if (!ctx) {
        return -1;
    }

    // Initialize encryption based on key length
    const EVP_CIPHER *cipher;
    switch (keylen) {
        case 16: cipher = EVP_aes_128_ecb(); break;
        case 24: cipher = EVP_aes_192_ecb(); break;
        case 32: cipher = EVP_aes_256_ecb(); break;
        default: 
            EVP_CIPHER_CTX_free(ctx);
            return -1;
    }

    if (EVP_EncryptInit_ex(ctx, cipher, NULL, (unsigned char *)key, NULL) != 1) {
        EVP_CIPHER_CTX_free(ctx);
        return -1;
    }

    // Encrypt the data
    if (EVP_EncryptUpdate(ctx, (unsigned char *)ciphertext, &ciphertext_len, 
                         (unsigned char *)plaintext, len) != 1) {
        EVP_CIPHER_CTX_free(ctx);
        return -1;
    }

    // Finalize the encryption
    if (EVP_EncryptFinal_ex(ctx, (unsigned char *)ciphertext + ciphertext_len, &final_len) != 1) {
        EVP_CIPHER_CTX_free(ctx);
        return -1;
    }

    EVP_CIPHER_CTX_free(ctx);
    return 0;
}

// Perform hardware crypto decryption
int hardware_crypto_decrypt(void *ciphertext, int len, void *key, int keylen, void *plaintext) {
    EVP_CIPHER_CTX *ctx;
    int plaintext_len;
    int final_len;

    // Create and initialize the context
    ctx = EVP_CIPHER_CTX_new();
    if (!ctx) {
        return -1;
    }

    // Initialize decryption based on key length
    const EVP_CIPHER *cipher;
    switch (keylen) {
        case 16: cipher = EVP_aes_128_ecb(); break;
        case 24: cipher = EVP_aes_192_ecb(); break;
        case 32: cipher = EVP_aes_256_ecb(); break;
        default: 
            EVP_CIPHER_CTX_free(ctx);
            return -1;
    }

    if (EVP_DecryptInit_ex(ctx, cipher, NULL, (unsigned char *)key, NULL) != 1) {
        EVP_CIPHER_CTX_free(ctx);
        return -1;
    }

    // Decrypt the data
    if (EVP_DecryptUpdate(ctx, (unsigned char *)plaintext, &plaintext_len, 
                         (unsigned char *)ciphertext, len) != 1) {
        EVP_CIPHER_CTX_free(ctx);
        return -1;
    }

    // Finalize the decryption
    if (EVP_DecryptFinal_ex(ctx, (unsigned char *)plaintext + plaintext_len, &final_len) != 1) {
        EVP_CIPHER_CTX_free(ctx);
        return -1;
    }

    EVP_CIPHER_CTX_free(ctx);
    return 0;
}

// Get NIC capabilities
int get_nic_capabilities(const char *ifname, int *features) {
    struct ifreq ifr;
    struct ethtool_value edata;
    int sock;
    int capabilities = 0;

    sock = socket(AF_INET, SOCK_DGRAM, 0);
    if (sock < 0) {
        return -1;
    }

    memset(&ifr, 0, sizeof(ifr));
    strncpy(ifr.ifr_name, ifname, IFNAMSIZ - 1);

    // Check various features
    int feature_tests[] = {
        ETHTOOL_GTXCSUM,  // TX checksum
        ETHTOOL_GRXCSUM,  // RX checksum
        ETHTOOL_GTSO,     // TSO
        ETHTOOL_GGSO,     // GSO
        ETHTOOL_GGRO,     // GRO
        ETHTOOL_GSG,      // Scatter-gather
        0, 0, 0, 0        // Placeholder for other features
    };

    for (int i = 0; i < 6 && feature_tests[i] != 0; i++) {
        edata.cmd = feature_tests[i];
        ifr.ifr_data = (caddr_t)&edata;
        
        if (ioctl(sock, SIOCETHTOOL, &ifr) == 0 && edata.data) {
            capabilities |= (1 << i);
        }
    }

    close(sock);
    *features = capabilities;
    return 0;
}