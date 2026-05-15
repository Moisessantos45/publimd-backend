package embeddings

import (
	"context"
	"fmt"
	"log"

	"time"
)

func TestGeneratePostEmbedding() {
	client := NewClient()

	post := NewPostEmbeddingRequest(12, "Ejemplo de post para generar embedding", "Este es un ejemplo de contenido de un post que se utilizará para generar un embedding. El embedding es una representación numérica del texto que captura su significado semántico.", []string{"Ejemplo"}, "Ejemplo")

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	resp, err := client.GeneratePostEmbedding(ctx, post)
	if err != nil {
		log.Fatal(err)
	}

	vector := ToPGVector(resp.Embedding)

	fmt.Println("Post ID:", resp.ID)
	fmt.Println("Dimensions:", resp.Dimensions)
	fmt.Println("Vector listo para pgvector:", vector[:120])
}
