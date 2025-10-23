# Guide de test avec MinIO (S3 local)

## Pourquoi MinIO ?
- 100% compatible S3 (même API qu'AWS/OVH)
- Gratuit et local (pas besoin de compte)
- Interface web pour voir vos fichiers
- Parfait pour tester avant de passer à OVH

## Étape 1 : Démarrer MinIO

```bash
# Démarrer le serveur MinIO
docker-compose -f docker-compose.minio.yml up -d

# Vérifier que c'est bien démarré
docker-compose -f docker-compose.minio.yml ps
```

## Étape 2 : Créer le bucket via l'interface web

1. Ouvrir votre navigateur : http://localhost:9001
2. Se connecter avec :
   - **Username:** minioadmin
   - **Password:** minioadmin123
3. Cliquer sur "Buckets" dans le menu de gauche
4. Cliquer sur "Create Bucket"
5. Nom du bucket : `mobula-backups`
6. Cliquer sur "Create Bucket"

## Étape 3 : Configurer votre application

```bash
# Utiliser la configuration MinIO
cp .env.minio.example .env

# Vérifier la configuration
cat .env | grep S3_
```

## Étape 4 : Tester l'upload

```bash
# Démarrer votre application normalement
make up

# Voir les logs
make logs
```

## Vérifier les uploads

1. Retourner sur http://localhost:9001
2. Cliquer sur "Buckets" > "mobula-backups"
3. Vous devriez voir la structure : `backups/ANNÉE/JOUR/MOIS/HEURE/`
4. Cliquer sur les fichiers .encrypted pour les voir

## Arrêter MinIO quand vous avez fini

```bash
docker-compose -f docker-compose.minio.yml down

# Pour tout supprimer (incluant les données)
docker-compose -f docker-compose.minio.yml down -v
```

## Passer à OVH plus tard

Quand vous aurez un compte OVH validé, il suffira de modifier le `.env` :

```bash
S3_ENABLED=true
S3_ENDPOINT=https://s3.gra.io.cloud.ovh.net
S3_REGION=gra
S3_ACCESS_KEY_ID=votre-clé-ovh
S3_SECRET_ACCESS_KEY=votre-secret-ovh
S3_BUCKET_NAME=votre-bucket-ovh
S3_BUCKET_PREFIX=backups
```

Tout le reste du code fonctionne exactement pareil !

## Alternatives à OVH (sans infos entreprise)

Si vous voulez tester un vrai cloud S3 sans contraintes :

1. **Scaleway** (Français) - https://console.scaleway.com
   - Compte facile à créer
   - Endpoint : `https://s3.fr-par.scw.cloud`

2. **Backblaze B2** - https://www.backblaze.com/b2/sign-up.html
   - 10 GB gratuits
   - Compatible S3

3. **Wasabi** - https://wasabi.com/
   - 1 TB gratuit pendant 30 jours
